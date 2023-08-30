package main

import (
	"bufio"
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

	layerAndDescriptorPaths := c.Generic("layer").(*flagutil.KeyValueFlag).List

	layerProvider := &blob.Index{
		Blobs: make(map[digest.Digest]string),
	}

	layerDescs := make([]ocispec.Descriptor, 0, len(layerAndDescriptorPaths))
	for _, layerAndDescriptorPath := range layerAndDescriptorPaths {
		layerDesc, err := ociutil.ReadDescriptorFromFile(layerAndDescriptorPath.Value)
		if err != nil {
			return err
		}

		layerProvider.Blobs[layerDesc.Digest] = layerAndDescriptorPath.Key
		layerDescs = append(layerDescs, layerDesc)
	}

	var entrypoint []string
	if entrypoint_file := c.String("entrypoint"); entrypoint_file != "" {
		var entrypointStruct struct {
			Entrypoint []string `json:"entrypoint"`
		}

		err := jsonutil.DecodeFromFile(entrypoint_file, &entrypointStruct)
		if err != nil {
			return fmt.Errorf("failed to read entrypoint config file: %w", err)
		}

		entrypoint = entrypointStruct.Entrypoint
	}

	log.Debugf("created descriptors for layers(n=%v): %#v", len(layerAndDescriptorPaths), layerDescs)

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
		entrypoint,
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
