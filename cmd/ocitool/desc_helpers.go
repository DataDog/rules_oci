package main

import (
	"context"
	"fmt"

	"github.com/DataDog/rules_oci/pkg/blob"
	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/containerd/containerd/content"
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
