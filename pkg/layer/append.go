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
	if annotations == nil {
		annotations = make(map[string]string)
	}

	manifest, err := ociutil.ImageManifestFromProvider(ctx, store, desc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", desc, err)
	}

	imageConfig, err := ociutil.ImageConfigFromProvider(ctx, store, manifest.Config)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image config (%v) in store: %w", manifest.Config, err)
	}

	baseRef := desc.Annotations[ocispec.AnnotationRefName]

	createdAnnotation := created.Format(time.RFC3339)
	annotations[ocispec.AnnotationCreated] = createdAnnotation

	// FIXME: add labels attribute to set this separately
	imageConfig.Config.Labels = annotations
	desc.Annotations = annotations
	manifest.Annotations = annotations
	imageConfig.Created = &created

	diffIDs := make([]digest.Digest, 0, len(layers))
	for _, layer := range layers {
		diffID, err := ociutil.GetLayerDiffID(ctx, store, layer)
		if err != nil {
			return ocispec.Descriptor{}, ocispec.Descriptor{},
				fmt.Errorf("failed to get diff ID of layer %q: %w", layer.Digest, err)
		}
		diffIDs = append(diffIDs, diffID)

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

	// we're OCI now
	desc.MediaType = ocispec.MediaTypeImageManifest
	manifest.MediaType = desc.MediaType
	// Append after we add the base image labels
	manifest.Layers = append(manifest.Layers, layers...)
	imageConfig.RootFS.DiffIDs = append(imageConfig.RootFS.DiffIDs, diffIDs...)

	newConfig, err := ociutil.IngestorJSONEncode(ctx, store, ocispec.MediaTypeImageConfig, imageConfig)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	manifest.Config = newConfig

	newManifest, err := ociutil.IngestorJSONEncode(ctx, store, ocispec.MediaTypeImageManifest, manifest)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	return newManifest, newConfig, nil
}
