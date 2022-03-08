package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DataDog/rules_oci/internal/flagutil"
	"github.com/DataDog/rules_oci/pkg/blob"
	"github.com/DataDog/rules_oci/pkg/layer"
	"github.com/DataDog/rules_oci/pkg/ociutil"

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

	baseDesc, err := ociutil.ReadDescriptorFromFile(c.String("base"))
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

		if ociutil.IsEmptyPlatform(manifestDesc.Platform) {
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

	log.WithField("base_desc", manifestDesc).Debugf("using as base")

	layerPaths := c.StringSlice("layer")

	layerProvider := &blob.Index{
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

	log.Debugf("created descriptors for layers(n=%v): %#v", len(layerPaths), layerDescs)

	outIngestor := layer.NewAppendIngester(c.String("out-manifest"), c.String("out-config"))

	newManifest, newConfig, err := layer.AppendLayers(
		c.Context,
		ociutil.SplitStore(outIngestor, ociutil.MultiProvider(allLocalProviders, layerProvider)),
		manifestDesc,
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
