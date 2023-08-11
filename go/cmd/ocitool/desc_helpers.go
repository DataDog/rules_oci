package main

import (
	"context"
	"fmt"

	"github.com/DataDog/rules_oci/go/pkg/blob"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	ErrNoResolvePlatform = fmt.Errorf("failed to resolve platform")
)

func FilePathsToDescriptors(ctx context.Context, descriptorPaths []string, resolvePlatforms bool, provider content.Provider) ([]ocispec.Descriptor, error) {
	descriptors := make([]ocispec.Descriptor, 0, len(descriptorPaths))

	for _, descPath := range descriptorPaths {
		desc, err := ociutil.ReadDescriptorFromFile(descPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load descriptors for index: %w", err)
		}

		// Only resolve if the platform is not defined
		if desc.Platform == nil || desc.Platform.OS == "" || desc.Platform.Architecture == "" {
			plat, err := ociutil.ResolvePlatformFromDescriptor(ctx, provider, desc)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrNoResolvePlatform, err)
			}

			desc.Platform = &plat
		}

		descriptors = append(descriptors, desc)
	}

	return descriptors, nil
}

func LoadLocalProviders(layoutPaths []string, relPath string) ([]content.Provider, error) {
	providers := make([]content.Provider, 0, len(layoutPaths))
	for _, path := range layoutPaths {
		provider, err := blob.LoadIndexFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load layout (%v): %w", path, err)
		}

		if relPath != "" {
			blobIdx := provider.(*blob.Index)

			provider, err = blobIdx.Rel(relPath)
			if err != nil {
				return nil, err
			}
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// loadManifestForImage loads the descriptor in the given file and returns the appropriate manifest for that descriptor
// and the given OS and architecture.
func loadManifestForImage(ctx context.Context, allLocalProviders content.Provider, descriptorFile string, os string, arch string) (ocispec.Descriptor, error) {
	// Read the descriptor, at this point we don't know if it's an image
	// manifest or index, so it's an unknown media type.
	unknownDesc, err := ociutil.ReadDescriptorFromFile(descriptorFile)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	targetPlatform := ocispec.Platform{
		OS:           os,
		Architecture: arch,
	}
	targetPlatformMatch := platforms.OnlyStrict(targetPlatform)

	// Resolve the unknown descriptor into an image manifest, if an index
	// match the requested platform.
	var manifestDesc ocispec.Descriptor
	if images.IsIndexType(unknownDesc.MediaType) {
		index, err := ociutil.ImageIndexFromProvider(ctx, allLocalProviders, unknownDesc)
		if err != nil {
			return ocispec.Descriptor{}, err
		}

		manifestDesc, err = ociutil.ManifestFromIndex(index, targetPlatformMatch)
		if err != nil {
			return ocispec.Descriptor{}, err
		}

		if !targetPlatformMatch.Match(*manifestDesc.Platform) {
			return ocispec.Descriptor{}, fmt.Errorf("invalid platform, expected %v, recieved %v", targetPlatform, *manifestDesc.Platform)
		}

	} else if images.IsManifestType(unknownDesc.MediaType) {
		manifestDesc = unknownDesc

		if ociutil.IsEmptyPlatform(manifestDesc.Platform) {
			platform, err := ociutil.ResolvePlatformFromDescriptor(ctx, allLocalProviders, manifestDesc)
			if err != nil {
				return ocispec.Descriptor{}, fmt.Errorf("no platform for base: %w", err)
			}

			manifestDesc.Platform = &platform
		}
	} else {
		return ocispec.Descriptor{}, fmt.Errorf("unknown base image type %q", unknownDesc.MediaType)
	}

	// Original comment: Copy the annotation with the original reference of the base image
	// so that we know when we push the image where those layers come from
	// for mount calls.
	//
	// This isn't true; we don't look at AnnotationRefName in ociutil.CopyContent
	if manifestDesc.Annotations == nil {
		manifestDesc.Annotations = make(map[string]string)
	}
	baseRef, ok := unknownDesc.Annotations[ocispec.AnnotationRefName]
	if ok {
		manifestDesc.Annotations[ocispec.AnnotationRefName] = baseRef
	}

	return manifestDesc, nil
}
