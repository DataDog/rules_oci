package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/pkg/blob"
	"github.com/DataDog/rules_oci/go/pkg/jsonutil"
	"github.com/DataDog/rules_oci/go/pkg/layer"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"

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

func AppendLayersCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

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

	allLocalProviders := ociutil.MultiProvider(localProviders...)

	// Read the base descriptor, at this point we don't know if it's a image
	// manifest or index, so it's an unknown media type.
	baseUnknownDesc, err := ociutil.ReadDescriptorFromFile(c.String("base"))
	if err != nil {
		return err
	}

	targetPlatform := ocispec.Platform{
		OS:           c.String("os"),
		Architecture: c.String("arch"),
	}
	targetPlatformMatch := platforms.Only(targetPlatform)

	// Resolve the unknown descriptor into an image manifest, if an index
	// match the requested platform.
	var baseManifestDesc ocispec.Descriptor
	if images.IsIndexType(baseUnknownDesc.MediaType) {
		index, err := ociutil.ImageIndexFromProvider(c.Context, allLocalProviders, baseUnknownDesc)
		if err != nil {
			return err
		}

		baseManifestDesc, err = ociutil.ManifestFromIndex(index, targetPlatformMatch)
		if err != nil {
			return err
		}

		if !targetPlatformMatch.Match(*baseManifestDesc.Platform) {
			return fmt.Errorf("invalid platform, expected %v, recieved %v", targetPlatform, *baseManifestDesc.Platform)
		}

	} else if images.IsManifestType(baseUnknownDesc.MediaType) {
		baseManifestDesc = baseUnknownDesc

		if ociutil.IsEmptyPlatform(baseManifestDesc.Platform) {
			platform, err := ociutil.ResolvePlatformFromDescriptor(c.Context, allLocalProviders, baseManifestDesc)
			if err != nil {
				return fmt.Errorf("no platform for base: %w", err)
			}

			baseManifestDesc.Platform = &platform
		}

	} else {
		return fmt.Errorf("Unknown base image type %q", baseUnknownDesc.MediaType)
	}

	// Original comment: Copy the annotation with the original reference of the base image
	// so that we know when we push the image where those layers come from
	// for mount calls.
	//
	// This isn't true; we don't look at AnnotationRefName in ociutil.CopyContent
	if baseManifestDesc.Annotations == nil {
		baseManifestDesc.Annotations = make(map[string]string)
	}
	baseRef, ok := baseUnknownDesc.Annotations[ocispec.AnnotationRefName]
	if ok {
		baseManifestDesc.Annotations[ocispec.AnnotationRefName] = baseRef
	}

	log.WithField("base_desc", baseManifestDesc).Debugf("using as base")

	tarPaths := c.StringSlice("tar")
	tarDescriptors := []ocispec.Descriptor{}
	for _, tarPath := range tarPaths {
		// Determine compression type of tarball
		compression, err := ociutil.DetectCompression(tarPath)
		if err != nil {
			return fmt.Errorf("failed to determine compression type for %s: %w", tarPath, err)
		}

		// Determine media type of tarball
		var mediaType string
		switch compression {
		case ociutil.CompressionGzip:
			mediaType = ocispec.MediaTypeImageLayerGzip
		case ociutil.CompressionZstd:
			mediaType = ocispec.MediaTypeImageLayerZstd
		case ociutil.CompressionNone:
			mediaType = ocispec.MediaTypeImageLayer
		}

		// Compute the sha256 of the tarball
		f, err := os.Open(tarPath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", tarPath, err)
		}
		defer f.Close()

		hasher := sha256.New()
		wc := ociutil.NewWriterCounter(hasher)
		if _, err := io.Copy(wc, f); err != nil {
			return fmt.Errorf("failed to read %s: %w", tarPath, err)
		}

		hash := hasher.Sum(nil)

		s := fmt.Sprintf("sha256:%x", hash)
		d, err := digest.Parse(s)
		if err != nil {
			return fmt.Errorf("failed to parse '%s' into a valid Digest: %w", s, err)
		}

		// Create descriptor
		desc := ocispec.Descriptor{
			Digest:    d,
			MediaType: mediaType,
			Size:      int64(wc.Count()),
		}

		tarDescriptors = append(tarDescriptors, desc)
	}

	layerAndDescriptorPaths := c.Generic("layer").(*flagutil.KeyValueFlag).List

	layerProvider := &blob.Index{
		Blobs: make(map[digest.Digest]string),
	}

	layerDescs := []ocispec.Descriptor{}
	for _, layerAndDescriptorPath := range layerAndDescriptorPaths {
		layerDesc, err := ociutil.ReadDescriptorFromFile(layerAndDescriptorPath.Value)
		if err != nil {
			return err
		}

		layerProvider.Blobs[layerDesc.Digest] = layerAndDescriptorPath.Key
		layerDescs = append(layerDescs, layerDesc)
	}

	// Treat raw tar files as if they were any other layer
	for i, tarPath := range tarPaths {
		tarDesc := tarDescriptors[i]
		layerProvider.Blobs[tarDesc.Digest] = tarPath
		layerDescs = append(layerDescs, tarDesc)
	}

	var cmd *[]string
	if cmdPath := c.String("cmd"); cmdPath != "" {
		var cmdStruct struct {
			Cmd []string `json:"cmd"`
		}
		err := jsonutil.DecodeFromFile(cmdPath, &cmdStruct)
		if err != nil {
			return fmt.Errorf("failed to read cmd config file: %w", err)
		}
		cmd = &cmdStruct.Cmd
	}

	var entrypoint *[]string
	if entrypointPath := c.String("entrypoint"); entrypointPath != "" {
		var entrypointStruct struct {
			Entrypoint []string `json:"entrypoint"`
		}
		err := jsonutil.DecodeFromFile(entrypointPath, &entrypointStruct)
		if err != nil {
			return fmt.Errorf("failed to read entrypoint config file: %w", err)
		}
		entrypoint = &entrypointStruct.Entrypoint
	}

	outIngestor := layer.NewAppendIngester(c.String("out-manifest"), c.String("out-config"))

	newManifest, newConfig, err := layer.AppendLayers(
		c.Context,
		ociutil.SplitStore(outIngestor, ociutil.MultiProvider(allLocalProviders, layerProvider)),
		baseManifestDesc,
		layerDescs,
		c.Generic("annotations").(*flagutil.KeyValueFlag).Map,
		c.Generic("labels").(*flagutil.KeyValueFlag).Map,
		c.StringSlice("env"),
		createdTimestamp,
		cmd,
		entrypoint,
		targetPlatform,
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
