package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/pkg/blob"
	"github.com/DataDog/rules_oci/pkg/ociutil"

	ocispecv "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func CreateIndexCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

	bi, err := blob.MergeIndex(localProviders...)
	if err != nil {
		return err
	}

	descriptorPaths := c.StringSlice("desc")

	descriptors, err := FilePathsToDescriptors(c.Context, descriptorPaths, true, bi)
	if err != nil {
		return err
	}
	log.WithField("manifests", descriptors).Debug("creating image index")

	idx := ocispec.Index{
		Versioned: ocispecv.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: descriptors,
	}

	// Save index to file and update descriptor
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
