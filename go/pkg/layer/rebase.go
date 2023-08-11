package layer

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images/converter"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// RebaseImage takes an original image, its current base image, and a new base image, and returns a new image with the old base image replaced with the new one.
func RebaseImage(ctx context.Context, store content.Store, originalImageDesc ocispec.Descriptor, oldBaseImageDesc ocispec.Descriptor, newBaseImageDesc ocispec.Descriptor, createdTimestamp time.Time) (ocispec.Descriptor, ocispec.Descriptor, error) {
	originalManifest, err := ociutil.ImageManifestFromProvider(ctx, store, originalImageDesc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", originalImageDesc, err)
	}

	originalImageConfig, err := ociutil.ImageConfigFromProvider(ctx, store, originalManifest.Config)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image config (%v) in store: %w", originalManifest.Config, err)
	}

	oldBaseManifest, err := ociutil.ImageManifestFromProvider(ctx, store, oldBaseImageDesc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", oldBaseImageDesc, err)
	}

	if len(oldBaseManifest.Layers) > len(originalManifest.Layers) {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("old base image (%v) has more layers than original image (%v)", oldBaseImageDesc, originalImageDesc)
	}

	for i, l := range oldBaseManifest.Layers {
		if l.Digest != originalManifest.Layers[i].Digest {
			return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("layer at index %d in old base image (%v) does not match layer in original image (%v)",
				i, oldBaseImageDesc, originalImageDesc)
		}
	}

	// Get the original image's layers after the old base image to append to the new base image and add the original image
	// as the OCI base image annotations to that layer, as is done for the layers from the base image in AppendLayers.
	var layersToAppend []ocispec.Descriptor
	origRef := originalImageDesc.Annotations[ocispec.AnnotationRefName]

	for _, origImgLayer := range originalManifest.Layers[len(oldBaseManifest.Layers):] {
		// If we have an original image ref, set the OCI base image annotations on the original image layers. This allows
		// ociutil.CopyContent to determine whether to copy the layer into the target repo via an OCI mount request.
		if origRef != "" {
			if origImgLayer.Annotations == nil {
				origImgLayer.Annotations = make(map[string]string)
			}
			if _, ok := origImgLayer.Annotations[ocispec.AnnotationBaseImageName]; !ok {
				origImgLayer.Annotations[ocispec.AnnotationBaseImageName] = origRef
			}

			if _, ok := origImgLayer.Annotations[ocispec.AnnotationBaseImageDigest]; !ok {
				origImgLayer.Annotations[ocispec.AnnotationBaseImageDigest] = originalImageDesc.Digest.String()
			}
		}

		origImgLayer.MediaType = converter.ConvertDockerMediaTypeToOCI(origImgLayer.MediaType)
		layersToAppend = append(layersToAppend, origImgLayer)
	}

	rebasedManifest, rebasedConfig, err := AppendLayers(ctx, store, newBaseImageDesc, layersToAppend, nil, originalImageConfig.Config.Labels, createdTimestamp, originalImageConfig.Config.Entrypoint)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("appending original image layers onto new base image: %w", err)
	}

	if rebasedManifest.Annotations == nil {
		rebasedManifest.Annotations = make(map[string]string)
	}

	// Set the new base image's name and digest as the values for the OCI base image annotations.
	rebasedManifest.Annotations[ocispec.AnnotationBaseImageName] = newBaseImageDesc.Annotations[ocispec.AnnotationRefName]
	rebasedManifest.Annotations[ocispec.AnnotationBaseImageDigest] = newBaseImageDesc.Digest.String()

	return rebasedManifest, rebasedConfig, nil
}
