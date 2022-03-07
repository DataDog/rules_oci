package layer

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/containerd/containerd/content"
	dreference "github.com/containerd/containerd/reference/docker"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func AppendLayers(ctx context.Context, store content.Store, desc ocispec.Descriptor, layers []ocispec.Descriptor, annotations map[string]string, created time.Time) (ocispec.Descriptor, ocispec.Descriptor, error) {
	manifest, err := ociutil.ImageManifestFromProvider(ctx, store, desc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", desc, err)
	}

	imageConfig, err := ociutil.ImageConfigFromProvider(ctx, store, manifest.Config)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image config (%v) in store: %w", manifest.Config, err)
	}

	baseRef := desc.Annotations[ocispec.AnnotationRefName]

	createdLabel := created.Format(time.RFC3339)
	annotations[ocispec.AnnotationCreated] = createdLabel

	imageConfig.Config.Labels = annotations
	desc.Annotations = annotations
	imageConfig.Created = &created

	// Get all of the digests of the layers to append to add to the diffids
	// in the image config
	digests := make([]digest.Digest, 0, len(layers))
	for _, layer := range layers {
		digests = append(digests, layer.Digest)
	}

	// Update image with base image reference
	if baseRef != "" {
		refTy, err := dreference.ParseNamed(baseRef)
		if err != nil {
			return ocispec.Descriptor{}, ocispec.Descriptor{}, err
		}

		ref := refTy.Name()

		for idx, layer := range manifest.Layers {
			if layer.Annotations == nil {
				layer.Annotations = make(map[string]string)
			}

			if _, ok := layer.Annotations[ociutil.AnnotationBaseImageName]; !ok {
				layer.Annotations[ociutil.AnnotationBaseImageName] = ref
			}

			if _, ok := layer.Annotations[ociutil.AnnotationBaseImageDigest]; !ok {
				layer.Annotations[ociutil.AnnotationBaseImageDigest] = desc.Digest.String()
			}

			manifest.Layers[idx] = layer
		}
	}

	manifest.MediaType = desc.MediaType
	// Append after we add the base image labels
	manifest.Layers = append(manifest.Layers, layers...)
	imageConfig.RootFS.DiffIDs = append(imageConfig.RootFS.DiffIDs, digests...)

	newConfig, err := ociutil.IngestorJSONEncode(ctx, store, manifest.Config.MediaType, imageConfig)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	manifest.Config = newConfig

	newManifest, err := ociutil.IngestorJSONEncode(ctx, store, desc.MediaType, manifest)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	return newManifest, newConfig, nil
}
