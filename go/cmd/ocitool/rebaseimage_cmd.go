package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/DataDog/rules_oci/go/pkg/blob"
	"github.com/DataDog/rules_oci/go/pkg/layer"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func RebaseImageCmd(c *cli.Context) error {
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

	originalDesc, err := loadManifestForImage(c.Context, allLocalProviders, c.String("original"), c.String("os"), c.String("arch"))
	if err != nil {
		return fmt.Errorf("loading descriptor for original image: %w", err)
	}
	log.WithField("original_desc", originalDesc).Debugf("using as original image")

	oldBaseDesc, err := loadManifestForImage(c.Context, allLocalProviders, c.String("old-base"), c.String("os"), c.String("arch"))
	if err != nil {
		return fmt.Errorf("loading descriptor for old base image: %w", err)
	}
	log.WithField("old_base_desc", oldBaseDesc).Debugf("using as old base image")

	newBaseDesc, err := loadManifestForImage(c.Context, allLocalProviders, c.String("new-base"), c.String("os"), c.String("arch"))
	if err != nil {
		return fmt.Errorf("loading descriptor for new base image: %w", err)
	}
	log.WithField("new_base_desc", newBaseDesc).Debugf("using as new base image")

	outIngestor := layer.NewAppendIngester(c.String("out-manifest"), c.String("out-config"))

	layerProvider := &blob.Index{
		Blobs: make(map[digest.Digest]string),
	}

	newManifest, newConfig, err := layer.RebaseImage(
		c.Context,
		ociutil.SplitStore(outIngestor, allLocalProviders),
		originalDesc,
		oldBaseDesc,
		newBaseDesc,
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
