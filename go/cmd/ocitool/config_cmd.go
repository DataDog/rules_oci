package main

import (
	"encoding/json"
	"fmt"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
	"os"
)

// ConfigCmd writes a given layouts config.
func ConfigCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), "")
	if err != nil {
		return err
	}

	allLocalProviders := ociutil.MultiProvider(localProviders...)

	// Read the base descriptor. Its unknown since we don't know if it's an image or index.
	baseUnknownDesc, err := ociutil.ReadDescriptorFromFile(c.String("base"))
	if err != nil {
		return err
	}

	targetPlatform := ocispec.Platform{
		OS:           c.String("os"),
		Architecture: c.String("arch"),
	}
	targetPlatformMatch := platforms.Only(targetPlatform)

	// Resolve the unknown descriptor into an image manifest
	// If the descriptor is an index, match the requested platform.
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
		return fmt.Errorf("unknown base image type %q", baseUnknownDesc.MediaType)
	}

	manifest, err := ociutil.ImageManifestFromProvider(c.Context, allLocalProviders, baseManifestDesc)
	if err != nil {
		return fmt.Errorf("no image manifest (%v) in store: %w", baseManifestDesc, err)
	}

	imageConfig, err := ociutil.ImageConfigFromProvider(c.Context, allLocalProviders, manifest.Config)
	if err != nil {
		return fmt.Errorf("no image config (%v) in store: %w", manifest.Config, err)
	}

	// Write the image config to the output file.
	outConfig, err := os.Create(c.String("out-config"))
	if err != nil {
		return err
	}
	defer func() { _ = outConfig.Close() }()

	err = json.NewEncoder(outConfig).Encode(imageConfig)
	if err != nil {
		return err
	}

	return nil
}
