// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"fmt"

	"github.com/containerd/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ManifestFromIndex gets an OCI Manifest from an image index that matches the
// desired platform.
func ManifestFromIndex(manifest ocispec.Index, platform platforms.MatchComparer) (ocispec.Descriptor, error) {
	for _, manifestDesc := range manifest.Manifests {
		if manifestDesc.Platform == nil {
			continue
		}

		if platform.Match(*manifestDesc.Platform) {
			return manifestDesc, nil
		}
	}

	return ocispec.Descriptor{}, fmt.Errorf("no matching manifest for platform")
}
