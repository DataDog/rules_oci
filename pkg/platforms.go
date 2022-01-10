package ociutil

import (
    "context"
    "fmt"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)


// imageConfigToPlatform creates a Platform from an Image Config type
func ImageConfigToPlatform(cfg ocispec.Image) ocispec.Platform {
	return ocispec.Platform{
		Architecture: cfg.Architecture,
		OS:           cfg.OS,
	}
}

// ResolvePlatformFromDescriptor resolves a platform from an image manifest, by
// looking at the OS and Arch variables in the image config
func ResolvePlatformFromDescriptor(ctx context.Context, provider content.Provider, desc ocispec.Descriptor) (ocispec.Platform, error) {
	if !images.IsManifestType(desc.MediaType) {
		return ocispec.Platform{}, fmt.Errorf("%w: media type not supported for platform resolution %v", errdefs.ErrFailedPrecondition, desc.MediaType)
	}

	imageManifest, err := ImageManifestFromProvider(ctx, provider, desc)
	if err != nil {
		return ocispec.Platform{}, err
	}

    imageConfig, err := ImageConfigFromProvider(ctx, provider, imageManifest.Config)
	if err != nil {
		return ocispec.Platform{}, err
	}

	platform := ImageConfigToPlatform(imageConfig)

	return platform, nil
}
