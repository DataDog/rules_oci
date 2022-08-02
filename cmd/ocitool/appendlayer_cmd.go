package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
    "context"

	"github.com/DataDog/rules_oci/internal/flagutil"
	"github.com/DataDog/rules_oci/pkg/blob"
	"github.com/DataDog/rules_oci/pkg/layer"
	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func loadStamp(r io.Reader) (map[string]string, error) {
	mp := make(map[string]string)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.Split(sc.Text(), " ")
		if len(line) < 2 {
			return nil, fmt.Errorf("failed to parse line: %v", sc.Text())
		}
		mp[line[0]] = line[1]
	}
	return mp, nil
}

func ResolveBaseImageForPlatform(ctx context.Context, baseUnknownDesc ocispec.Descriptor, targetPlatform ocispec.Platform, provider content.Provider) (ocispec.Descriptor, error) {

	targetPlatformMatch := platforms.Only(targetPlatform)

	// Resolve the unknown descriptor into an image manifest, if an index
	// match the requested platform.
	var baseManifestDesc ocispec.Descriptor
	if images.IsIndexType(baseUnknownDesc.MediaType) {
		index, err := ociutil.ImageIndexFromProvider(ctx, provider, baseUnknownDesc)
		if err != nil {
			return ocispec.Descriptor{}, err
		}

		baseManifestDesc, err = ociutil.ManifestFromIndex(index, targetPlatformMatch)
		if err != nil {
			return  ocispec.Descriptor{}, err
		}

		if !targetPlatformMatch.Match(*baseManifestDesc.Platform) {
			return ocispec.Descriptor{}, fmt.Errorf("invalid platform, expected %v, recieved %v", targetPlatform, *baseManifestDesc.Platform)
		}

	} else if images.IsManifestType(baseUnknownDesc.MediaType) {
		baseManifestDesc = baseUnknownDesc

		if ociutil.IsEmptyPlatform(baseManifestDesc.Platform) {
			platform, err := ociutil.ResolvePlatformFromDescriptor(ctx, provider, baseManifestDesc)
			if err != nil {
				return ocispec.Descriptor{}, fmt.Errorf("no platform for base: %w", err)
			}

			baseManifestDesc.Platform = &platform
		}

	} else {
		return ocispec.Descriptor{}, fmt.Errorf("Unknown base image type %q", baseUnknownDesc.MediaType)
	}

	// Copy the annotation with the original reference of the base image
	// so that we know when we push the image where those layers come from
	// for mount calls.
	if baseManifestDesc.Annotations == nil {
		baseManifestDesc.Annotations = make(map[string]string)
	}
	baseRef, ok := baseUnknownDesc.Annotations[ocispec.AnnotationRefName]
	if ok {
		baseManifestDesc.Annotations[ocispec.AnnotationRefName] = baseRef
	}

    return baseManifestDesc, nil
}

func CreateIndexForBlobFiles(blobPaths ...string) (*blob.Index, []ocispec.Descriptor, error) {
	blobProvider := &blob.Index{
		Blobs: make(map[digest.Digest]string),
	}

	blobDescs := make([]ocispec.Descriptor, 0, len(blobPaths))
	for _, lp := range blobPaths {
		ld, reader, err := ociutil.CreateDescriptorFromFile(lp)
		if err != nil {
			return nil, nil, err
		}
		reader.Close()
		ld.MediaType = ocispec.MediaTypeImageLayerGzip

		blobProvider.Blobs[ld.Digest] = lp
		blobDescs = append(blobDescs, ld)
	}

    return blobProvider, blobDescs, nil
}

func AppendLayersCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}
	allLocalProviders := ociutil.MultiProvider(localProviders...)

	var stampVars map[string]string
	bazelVersionFilePath := c.String("bazel-version-file")
	if bazelVersionFilePath != "" {
		file, err := os.Open(bazelVersionFilePath)
		if err != nil {
			return err
		}

		stampVars, err = loadStamp(file)
		file.Close()
		if err != nil {
			return err
		}
	}

	createdTimestamp := time.Unix(0, 0)
	if timeStr, ok := stampVars["BUILD_TIMESTAMP"]; ok {
		timeInt, err := strconv.ParseInt(timeStr, 10, 0)
		if err != nil {
			return err
		}

		createdTimestamp = time.Unix(timeInt, 0)
	}

	targetPlatform := ocispec.Platform{
		OS:           c.String("os"),
		Architecture: c.String("arch"),
	}


	layerPaths := c.StringSlice("layer")
    layerProvider, layerDescs, err := CreateIndexForBlobFiles(layerPaths...)
    if err != nil {
        return err
    }

	log.Debugf("created descriptors for layers(n=%v): %#v", len(layerPaths), layerDescs)

    baseDescriptorPath := c.String("base")
    var baseManifestDesc ocispec.Descriptor
    if baseDescriptorPath != "" {
        // Read the base descriptor, at this point we don't know if it's a image
        // manifest or index, so it's an unknown media type.
        baseUnknownDesc, err := ociutil.ReadDescriptorFromFile(baseDescriptorPath)
        if err != nil {
            return err
        }

        log.WithField("base_desc", baseUnknownDesc).Debugf("found base descriptor, resolving platform-specific base")

        baseManifestDesc, err = ResolveBaseImageForPlatform(c.Context, baseUnknownDesc, targetPlatform, allLocalProviders)
        if err != nil {
            return err
        }

        log.WithField("base_desc", baseManifestDesc).Debugf("resolved final base")
    } else {
        return fmt.Errorf("don't support oci_image without base yet")
    }

	outIngestor := layer.NewAppendIngester(c.String("out-manifest"), c.String("out-config"))
	newManifest, newConfig, err := layer.AppendLayers(
		c.Context,
		ociutil.SplitStore(outIngestor, ociutil.MultiProvider(allLocalProviders, layerProvider)),
		baseManifestDesc,
		layerDescs,
		c.Generic("annotations").(*flagutil.KeyValueFlag).Map,
		createdTimestamp,
	)
	if err != nil {
		return err
	}

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
