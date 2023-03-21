package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/go/pkg/blob"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"

	ocispecv "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func CreateImageManifestCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

	bi, err := blob.MergeIndex(localProviders...)
	if err != nil {
		return err
	}

	configDesc, err := ociutil.ReadDescriptorFromFile(c.String("config-desc"))
	if err != nil {
		return err
	}

	descriptorPaths := c.StringSlice("layer-desc")
	descriptors, err := FilePathsToDescriptors(c.Context, descriptorPaths, true, bi)
	if err != nil {
		return err
	}
	log.WithField("manifests", descriptors).Debug("creating image manifest")

	manifest := ocispec.Manifest{
		Versioned: ocispecv.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    descriptors,
	}

	desc, err := ociutil.CopyJSONToFileAndCreateDescriptor(&manifest, c.String("out-manifest"))
	if err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	// Append image index to blob index
	bi.Blobs[desc.Digest] = c.String("out-manifest")

	err = bi.WriteToFile(c.String("out-layout"))
	if err != nil {
		return err
	}

	return nil
}
