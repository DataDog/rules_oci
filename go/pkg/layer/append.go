package layer

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/rules_oci/go/pkg/ociutil"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images/converter"
	dreference "github.com/containerd/containerd/reference/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// FIXME: this was removed from ocispec in https://github.com/opencontainers/image-spec/pull/1078 as
// being a redundant duplicate of "org.opencontainers.image.created"; the code should be updated to
// use that.
const AnnotationArtifactDescription = "org.opencontainers.artifact.created"

func AppendLayers(ctx context.Context, store content.Store, baseManifestDesc ocispec.Descriptor, layers []ocispec.Descriptor, annotations map[string]string, labels map[string]string, env []string, created time.Time, entrypoint []string) (ocispec.Descriptor, ocispec.Descriptor, error) {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	manifest, err := ociutil.ImageManifestFromProvider(ctx, store, baseManifestDesc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", baseManifestDesc, err)
	}

	imageConfig, err := ociutil.ImageConfigFromProvider(ctx, store, manifest.Config)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image config (%v) in store: %w", manifest.Config, err)
	}

	baseRef := baseManifestDesc.Annotations[ocispec.AnnotationRefName]

	createdAnnotation := created.Format(time.RFC3339)
	annotations[ocispec.AnnotationCreated] = createdAnnotation

	if imageConfig.Config.Labels == nil {
		imageConfig.Config.Labels = make(map[string]string)
	}
	for label, value := range labels {
		if value == "" {
			delete(imageConfig.Config.Labels, label)
		} else {
			imageConfig.Config.Labels[label] = value
		}
	}

	baseManifestDesc.Annotations = annotations
	manifest.Annotations = annotations
	imageConfig.Created = &created

	diffIDs := make([]digest.Digest, 0, len(layers))
	history := make([]ocispec.History, 0, len(layers))
	for _, layer := range layers {
		diffID, err := ociutil.GetLayerDiffID(ctx, store, layer)
		if err != nil {
			return ocispec.Descriptor{}, ocispec.Descriptor{},
				fmt.Errorf("failed to get diff ID of layer %q: %w", layer.Digest, err)
		}
		diffIDs = append(diffIDs, diffID)

		// Using Comment as the thing-that-created-the-layer and CreatedBy as the
		// source-of-the-layer apes what docker does.
		layerHistory := ocispec.History{
			Created: &created,
			Comment: "rules_oci",
		}
		if description, ok := layer.Annotations[AnnotationArtifactDescription]; ok {
			layerHistory.CreatedBy = description
		}
		history = append(history, layerHistory)
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

			// It's arguably incorrect to label the layers with the base image name/digest, since
			// that annotation is intended to indicate the image which an image builds on, not the
			// image origin of the layer it comes from; see
			// https://github.com/opencontainers/image-spec/issues/821.  This code only annotates
			// layers with no annotations, but unannotated layers are the default with
			// docker-created images, and we cannot know whether _all_ the layers in those images
			// _really_ came FROM the specified base layer.  Only if all images are built with
			// rules_oci can we guarantee this.
			//
			// SIDE EFFECT: The presence of this label is used in ociutil.CopyContent to determine
			// whether to copy the layer into the target repo via an OCI mount request i.e. we use
			// the label to tag layers that should already exist in the target registry.
			if _, ok := layer.Annotations[ocispec.AnnotationBaseImageName]; !ok {
				layer.Annotations[ocispec.AnnotationBaseImageName] = ref
			}

			if _, ok := layer.Annotations[ocispec.AnnotationBaseImageDigest]; !ok {
				layer.Annotations[ocispec.AnnotationBaseImageDigest] = baseManifestDesc.Digest.String()
			}

			layer.MediaType = converter.ConvertDockerMediaTypeToOCI(layer.MediaType)

			manifest.Layers[idx] = layer
		}
	}

	// we're OCI now
	baseManifestDesc.MediaType = ocispec.MediaTypeImageManifest
	manifest.MediaType = baseManifestDesc.MediaType
	// Append after we add the base image labels
	manifest.Layers = append(manifest.Layers, layers...)
	imageConfig.RootFS.DiffIDs = append(imageConfig.RootFS.DiffIDs, diffIDs...)
	imageConfig.History = append(imageConfig.History, history...)

	imageConfig.Author = "rules_oci"
	imageConfig.Config.Entrypoint = entrypoint
	imageConfig.Config.Env = append(imageConfig.Config.Env, env...)

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
