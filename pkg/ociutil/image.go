// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"context"

	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ImageIndexFromProvider fetches an image index from a provider and decodes
// it.
//
// XXX: This function assumes that the data is json encoded.
func ImageIndexFromProvider(ctx context.Context, provider content.Provider, desc ocispec.Descriptor) (ocispec.Index, error) {
	var index ocispec.Index

	err := ProviderJSONDecode(ctx, provider, desc, &index)
	if err != nil {
		return ocispec.Index{}, err
	}

	return index, nil
}

// ImageManifestFromProvider fetches an image manifest from a provider and decodes
// it.
//
// XXX: This function assumes that the data is json encoded.
func ImageManifestFromProvider(ctx context.Context, provider content.Provider, desc ocispec.Descriptor) (ocispec.Manifest, error) {
	var cfg ocispec.Manifest

	err := ProviderJSONDecode(ctx, provider, desc, &cfg)
	if err != nil {
		return ocispec.Manifest{}, err
	}

	return cfg, nil
}

// ImageConfigFromProvider fetches an image config from a provider and decodes
// it. The descriptor must point directly at the image config.
//
// XXX: This function assumes that the data is json encoded.
func ImageConfigFromProvider(ctx context.Context, provider content.Provider, desc ocispec.Descriptor) (ocispec.Image, error) {
	var cfg ocispec.Image

	err := ProviderJSONDecode(ctx, provider, desc, &cfg)
	if err != nil {
		return ocispec.Image{}, err
	}

	return cfg, nil
}
