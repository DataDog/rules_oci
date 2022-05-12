package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	ocispecv "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
)

func PublishRulesCmd(c *cli.Context) error {
	resolver := ociutil.DefaultResolver()

	desc, err := resolver.PushBlob(c.Context, c.String("file"), c.String("ref"), "application/x.datadog.bazel.rules.tar+gz")
	if err != nil {
		return fmt.Errorf("failed to push blob: %w", err)
	}

	manifest := ocispec.Manifest{
		Versioned: ocispecv.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    desc,
		Annotations: map[string]string{
			"org.opencontainers.image.source": "github.com/DataDog/rules_oci",
		},
	}

	_, err = resolver.MarshalAndPushContent(c.Context, c.String("ref"), manifest, manifest.MediaType)
	if err != nil {
		return err
	}

	url, err := ociutil.DescriptorToURL(c.String("ref"), desc)
	if err != nil {
		return err
	}

	fmt.Printf("Pushed: %v", url)

	return nil
}
