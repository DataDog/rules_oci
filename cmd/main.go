package main

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/opencontainers/go-digest"
	ocispecv "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	orascontent "oras.land/oras-go/pkg/content"
	"oras.land/oras-go/pkg/oras"
)

var app = &cli.App{
	Name: "ocitool",
	Before: func(c *cli.Context) error {
		log.SetLevel(log.InfoLevel)

		if c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	},
	Commands: []*cli.Command{
		{
			Name:   "pull",
			Usage:  "Pull an OCI artifact",
			Action: PullCmd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "shallow",
					Usage: "Pull only the top level manifests.",
					Value: false,
				},
			},
		},
		{
			Name:   "generate-build-files",
			Action: GenerateBuildFilesCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "image-digest",
				},
			},
		},
		{
			Name:   "create-layer",
			Action: CreateLayerCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "dir",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name: "file",
				},
				&cli.StringFlag{
					Name: "out",
				},
				&cli.StringFlag{
					Name: "outd",
				},
				&cli.GenericFlag{
					Name:  "symlink",
					Value: &KeyValueFlag{},
				},
			},
		},
		{
			Name:   "append-layers",
			Action: AppendLayersCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "base",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name: "layer",
				},
				&cli.StringFlag{
					Name: "outd",
				},
				&cli.StringFlag{
					Name: "os",
				},
				&cli.StringFlag{
					Name: "arch",
				},
				&cli.StringFlag{
					Name: "out-manifest",
				},
				&cli.StringFlag{
					Name: "out-config",
				},
				&cli.StringFlag{
					Name: "out-layout",
				},
			},
		},
		{
			Name:   "push",
			Action: PushCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "layout-relative",
				},
				&cli.StringFlag{
					Name: "desc",
				},
				&cli.StringFlag{
					Name: "target-ref",
				},
			},
		},
		{
			Name:   "digest",
			Action: DigestCmd,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name: "desc",
				},
				&cli.StringFlag{
					Name: "out",
				},
			},
		},
		{
			Name:   "create-index",
			Action: CreateIndexCmd,
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name: "desc",
				},
				&cli.StringFlag{
					Name: "out-index",
				},
				&cli.StringFlag{
					Name: "out-layout",
				},
				&cli.StringFlag{
					Name: "outd",
				},
			},
		},
	},
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "debug",
			Value: false,
		},
		&cli.StringSliceFlag{
			Name:     "layout",
			Usage:    "Filepath to a directory with the OCI Layout structure, if it doesn't exist it creates a new directory",
			Required: false,
		},
		&cli.UintFlag{
			Name:  "parallel",
			Usage: "Parallelism of pushing/pulling operations",
			Value: 5,
		},
	},
}

func appendFileToTarWriter(filePath string, basedir string, tw *tar.Writer) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}

	hdr.ChangeTime = time.Time{}
	hdr.ModTime = time.Time{}
	hdr.AccessTime = time.Time{}

	hdr.Name = filepath.Join(basedir, filepath.Base(fi.Name()))

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}

func getLocalProviders(c *cli.Context) ([]content.Provider, error) {
	paths := c.StringSlice("layout")

	providers := make([]content.Provider, 0, len(paths))
	for _, path := range paths {
		provider, err := ociutil.LoadBlobIndex(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load layout (%v): %w", path, err)
		}

		if relPath := c.String("layout-relative"); relPath != "" {
			blobIdx := provider.(*ociutil.BlobIndex)

			provider, err = blobIdx.Rel(relPath)
			if err != nil {
				return nil, err
			}
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

func DigestCmd(c *cli.Context) error {
	desc, err := ociutil.DescriptorFromFile(c.String("desc"))
	if err != nil {
		return err
	}

	err = os.WriteFile(c.String("out"), []byte(desc.Digest.String()), 0655)
	if err != nil {
		return err
	}

	return nil
}

func PushCmd(c *cli.Context) error {
	localProviders, err := getLocalProviders(c)
	if err != nil {
		return err
	}

	allProviders := ociutil.MultiProvider(localProviders...)

	baseDesc, err := ociutil.DescriptorFromFile(c.String("desc"))
	if err != nil {
		return err
	}

	resolver := ociutil.NewDDRegistryResolver()

	ref := c.String("target-ref")

	pusher, err := resolver.Pusher(c.Context, ref)
	if err != nil {
		return err
	}

	regIng, ok := pusher.(content.Ingester)
	if !ok {
		return fmt.Errorf("pusher not an ingester: %T", pusher)
	}

	imagesHandler := images.ChildrenHandler(allProviders)

	err = ociutil.CopyContentFromHandler(c.Context, imagesHandler, allProviders, regIng, baseDesc)
	if err != nil {
		return err
	}

	fmt.Printf("Reference: %v@%v\n", ref, baseDesc.Digest.String())

	return nil
}

type KeyValueFlag struct {
	m map[string]string
}

func (k *KeyValueFlag) String() string {
	return ""
}

func (k *KeyValueFlag) Set(value string) error {
	if k.m == nil {
		k.m = make(map[string]string)
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) < 2 {
		return fmt.Errorf("not a valid mapping, must be k=k: %v", value)
	}

	k.m[parts[0]] = parts[1]
	return nil
}

func CreateLayerCmd(c *cli.Context) error {
	dir := c.String("dir")
	files := c.StringSlice("file")

	out, err := os.Create(c.String("out"))
	if err != nil {
		return err
	}

	digester := digest.SHA256.Digester()
	wc := ociutil.NewWriterCounter(io.MultiWriter(out, digester.Hash()))
	tw := tar.NewWriter(wc)
	defer tw.Close()

	for _, filePath := range files {
		err = appendFileToTarWriter(filePath, dir, tw)
		if err != nil {
			return err
		}
	}

	for k, v := range c.Generic("symlink").(*KeyValueFlag).m {
		err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     k,
			Linkname: v,
		})
		if err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	// Need to flush before we count bytes and digest, might as well close since
	// it's not needed anymore.
	tw.Close()

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Size:      int64(wc.Count()),
		Digest:    digester.Digest(),
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	return nil
}

func AppendLayersCmd(c *cli.Context) error {
	localProviders, err := getLocalProviders(c)
	if err != nil {
		return err
	}

	allLocalProviders := ociutil.MultiProvider(localProviders...)

	baseDesc, err := ociutil.DescriptorFromFile(c.String("base"))
	if err != nil {
		return err
	}

	targetPlatform := ocispec.Platform{
		OS:           c.String("os"),
		Architecture: c.String("arch"),
	}
	targetPlatformMatch := platforms.Only(targetPlatform)

	var manifestDesc ocispec.Descriptor
	if images.IsIndexType(baseDesc.MediaType) {
		index, err := ociutil.ImageIndexFromProvider(c.Context, allLocalProviders, baseDesc)
		if err != nil {
			return err
		}

		manifestDesc, err = ociutil.ManifestFromIndex(index, targetPlatformMatch)
		if err != nil {
			return err
		}
	} else if images.IsManifestType(baseDesc.MediaType) {
		manifestDesc = baseDesc

		if manifestDesc.Platform == nil || manifestDesc.Platform.Architecture == "" || manifestDesc.Platform.OS == "" {
			platform, err := ociutil.ResolvePlatformFromDescriptor(c.Context, allLocalProviders, manifestDesc)
			if err != nil {
				return fmt.Errorf("no platform for base: %w", err)
			}

			manifestDesc.Platform = &platform
		}

	} else {
		return fmt.Errorf("Unknown base image type %q", baseDesc.MediaType)
	}

	if !targetPlatformMatch.Match(*manifestDesc.Platform) {
		return fmt.Errorf("invalid platform, expected %v, recieved %v", targetPlatform, *manifestDesc.Platform)
	}

	baseRef, ok := baseDesc.Annotations[ocispec.AnnotationRefName]
	if ok {
		if manifestDesc.Annotations == nil {
			manifestDesc.Annotations = make(map[string]string)
		}

		manifestDesc.Annotations[ocispec.AnnotationRefName] = baseRef
	}

	log.Printf("using %#v as base", manifestDesc)

	layerPaths := c.StringSlice("layer")

	layerProvider := &ociutil.BlobIndex{
		Blobs: make(map[digest.Digest]string),
	}

	layerDescs := make([]ocispec.Descriptor, 0, len(layerPaths))
	for _, lp := range layerPaths {
		ld, reader, err := ociutil.CreateDescriptorFromFile(lp)
		if err != nil {
			return err
		}
		reader.Close()
		ld.MediaType = images.MediaTypeDockerSchema2Layer

		layerProvider.Blobs[ld.Digest] = lp
		layerDescs = append(layerDescs, ld)
	}

	log.Printf("created descriptors for layers(n=%v): %#v", len(layerPaths), layerDescs)

	outIngestor := ociutil.NewAppendLayerIngestor(c.String("out-manifest"), c.String("out-config"))

	newManifest, newConfig, err := ociutil.AppendLayers(
		c.Context,
		ociutil.SplitStore(outIngestor, ociutil.MultiProvider(allLocalProviders, layerProvider)),
		manifestDesc,
		layerDescs,
	)
	if err != nil {
		return err
	}

	log.Printf("appended layers!")

	layerProvider.Blobs[newManifest.Digest] = c.String("out-manifest")
	layerProvider.Blobs[newConfig.Digest] = c.String("out-config")

	err = layerProvider.WriteToFile(c.String("out-layout"))
	if err != nil {
		return err
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), newManifest)
	if err != nil {
		return err
	}

	return nil
}

func GenerateBuildFilesCmd(c *cli.Context) error {
	allLocalLayoutsPaths := c.StringSlice("layout")
	if len(allLocalLayoutsPaths) > 1 {
		return fmt.Errorf("too many layouts")
	} else if len(allLocalLayoutsPaths) <= 0 {
		return fmt.Errorf("need at least one layout")
	}

	layoutRootPath := allLocalLayoutsPaths[0]

	layout, err := orascontent.NewOCI(layoutRootPath)
	if err != nil {
		return err
	}

	refs := layout.ListReferences()
	refDescs := make([]ocispec.Descriptor, 0, len(refs))

	for _, r := range refs {
		refDescs = append(refDescs, r)
	}

	log.Debugf("Layout root: %#v", refs)

	err = images.Walk(
		context.Background(),
		ociutil.GenerateBuildFilesHandler(images.ChildrenHandler(layout), layoutRootPath, layout),
		refDescs...,
	)
	if err != nil {
		return err
	}

	imageTargetDigest := c.String("image-digest")
	if imageTargetDigest != "" {
		err = os.MkdirAll(filepath.Join(layoutRootPath, "image"), 0700)
		if err != nil {
			return err
		}

		imageTargetBuildFilePath := filepath.Join(layoutRootPath, "image", "BUILD.bazel")
		imageTargetBuild := rule.EmptyFile(imageTargetBuildFilePath, "")

		aliasRule := rule.NewRule("alias", "image")
		aliasRule.SetAttr("actual", dgstToManifestLabel(digest.Digest(imageTargetDigest)))
		aliasRule.SetAttr("visibility", ociutil.PublicVisibility)
		aliasRule.Insert(imageTargetBuild)

		err = imageTargetBuild.Save(imageTargetBuildFilePath)
		if err != nil {
			return err
		}

		log.Debugf("Created BUILD file in image package")
	}

	log.Debugf("Done generating build files")

	return nil
}

// TODO redeclared a couple other places
func dgstToManifestLabel(dgst digest.Digest) string {
	return fmt.Sprintf("//blobs/%s:%s", dgst.Algorithm().String(), dgstToManifestLabelName(dgst))
}

func dgstToManifestLabelName(dgst digest.Digest) string {
	return fmt.Sprintf("%v-%v-%v", "manifest", dgst.Algorithm().String(), dgst.Encoded())
}

func CreateIndexCmd(c *cli.Context) error {
	localProviders, err := getLocalProviders(c)
	if err != nil {
		return err
	}

	bi, err := ociutil.MergeProviders(localProviders...)
	if err != nil {
		return err
	}

	descriptorPaths := c.StringSlice("desc")
	descriptors := make([]ocispec.Descriptor, 0, len(descriptorPaths))

	for _, descPath := range descriptorPaths {
		desc, err := ociutil.DescriptorFromFile(descPath)
		if err != nil {
			return fmt.Errorf("failed to load descriptors for index: %w", err)
		}

		// Only resolve if the platform is not defined
		if desc.Platform == nil || desc.Platform.OS == "" || desc.Platform.Architecture == "" {
			plat, err := ociutil.ResolvePlatformFromDescriptor(c.Context, bi, desc)
			if err != nil {
				return fmt.Errorf("failed to resolve platform for manifest: %w", err)
			}

			desc.Platform = &plat
		}

		descriptors = append(descriptors, desc)
	}

	log.WithField("manifests", descriptors).Debug("creating image index")

	idx := ocispec.Index{
		Versioned: ocispecv.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: descriptors,
	}

	desc, err := ociutil.CopyJSONToFileAndCreateDescriptor(&idx, c.String("out-index"))
	if err != nil {
		return fmt.Errorf("failed to write index: %w", err)
	}

	desc.MediaType = ocispec.MediaTypeImageIndex

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	// Append image index to blob index
	bi.Blobs[desc.Digest] = c.String("out-index")

	err = bi.WriteToFile(c.String("out-layout"))
	if err != nil {
		return err
	}

	return nil
}

func PullCmd(c *cli.Context) error {
	res := ociutil.NewDDRegistryResolver()

	ref := c.Args().First()

	name, desc, err := res.Resolve(c.Context, ref)
	if err != nil {
		return err
	}

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}

	desc.Annotations[ocispec.AnnotationRefName] = name

	log.Printf("Resolved descriptor for %v: %#v", name, desc)

	layoutPath := c.StringSlice("layout")[0]

	layout, err := orascontent.NewOCI(layoutPath)
	if err != nil {
		return err
	}

	log.Printf("Found layout at '%v'", layoutPath)

	remoteFetcher, err := res.Fetcher(c.Context, name)
	if err != nil {
		return err
	}

	log.Printf("Created fetcher for %v", ref)

	provider := &oras.ProviderWrapper{
		Fetcher: remoteFetcher,
	}

	sem := semaphore.NewWeighted(1)

	imagesHandler := images.ChildrenHandler(provider)
	if c.Bool("shallow") {
		imagesHandler = ociutil.ContentTypesFilterHandler(imagesHandler,
			ocispec.MediaTypeImageManifest,
			ocispec.MediaTypeImageIndex,
			ocispec.MediaTypeImageConfig,
			images.MediaTypeDockerSchema2Manifest,
			images.MediaTypeDockerSchema2ManifestList,
			images.MediaTypeDockerSchema2Config,
		)
	}

	err = images.Dispatch(c.Context, ociutil.CopyContentHandler(imagesHandler, provider, layout), sem, desc)
	if err != nil {
		return err
	}

	layout.AddReference(name, desc)
	err = layout.SaveIndex()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	app.RunAndExitOnError()
}
