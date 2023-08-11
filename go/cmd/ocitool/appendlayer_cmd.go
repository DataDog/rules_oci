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

	baseManifestDesc, err := loadManifestForImage(c.Context, allLocalProviders, c.String("base"), c.String("os"), c.String("arch"))
	if err != nil {
		return fmt.Errorf("loading descriptor for base image: %w", err)
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
