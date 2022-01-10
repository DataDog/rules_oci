package ociutil

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	dreference "github.com/containerd/containerd/reference/docker"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

// TODO use upstream defs when a new release is cut
// https://github.com/opencontainers/image-spec/commit/71ccc68078c473544315863eabb2f95140f7e1bf#diff-05a9698dc79be9f08ba5b6fbbaa6bb013a61c3b2db9b5cd1aa570677f7065c0c
var (
	// AnnotationBaseImageDigest is the annotation key for the digest of the image's base image.
	AnnotationBaseImageDigest = "org.opencontainers.image.base.digest"

	// AnnotationBaseImageName is the annotation key for the image reference of the image's base image.
	AnnotationBaseImageName = "org.opencontainers.image.base.name"
)

// AddLayers will download the given tar files from s3, add them as layers onto the base
// image and push the resulting image to the target.
func (resolver Resolver) AddLayers(ctx context.Context, baseRef, targetRef, bucket, region, goarch string, keys, urls []string) (digest.Digest, error) {
	///////////
	// Fetch base image details from ref (i.e. registry.ddbuild.io/base:bionic-multi-arch)
	///////////
	_, baseDesc, err := resolver.Resolve(ctx, baseRef)
	if err != nil {
		return "", fmt.Errorf("unable to resolve base image %q: %w", baseRef, err)
	}
	log.WithField("descriptor", baseDesc).WithField("ref", baseRef).
		Debugf("resolved base manifest")

	manifest, image, err := resolver.GetManifestAndConfigImage(ctx, baseRef, baseDesc, goarch)
	if err != nil {
		return "", fmt.Errorf("unable to get base image manifest %q: %w",
			baseRef, err)
	}
	log.WithField("manifest", manifest).WithField("ref", baseRef).
		Debug("resolved manifest for base image")

	///////////
	// Push all base layers to target ref
	///////////
	err = resolver.MountBlobs(ctx, baseDesc, baseRef, targetRef, goarch)
	if err != nil {
		return "", fmt.Errorf("unable to mount base blobs to target: %w", err)
	}
	log.WithField("base", baseRef).WithField("target", targetRef).
		Debug("mounted layers from base to target")

	///////////
	// Download files via URLs, create digest and push them to the registry
	///////////
	layers, digests, err := resolver.DownloadAndPushBlobs(ctx, bucket, region, keys, targetRef)
	if err != nil {
		return "", fmt.Errorf("unable to download and push layers: %w", err)
	}
	log.Debug("all file URLs have been downloaded and pushed")

	///////////
	// Modify Image Manifest and Image Config with layers of tar files
	///////////
	manifest.Layers = append(manifest.Layers, layers...)
	// note: we're only able to use the digest for the diff ID b/c the layers we're adding
	// are _uncompressed_.
	image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, digests...)

	layers, digests, err = resolver.ConvertDebsAndPushBlobs(ctx, targetRef, urls)
	if err != nil {
		return "", fmt.Errorf("unable to convert debs and push: %w", err)
	}
	manifest.Layers = append(manifest.Layers, layers...)
	image.RootFS.DiffIDs = append(image.RootFS.DiffIDs, digests...)

	///////////
	// Push the image config
	///////////
	newDesc, err := resolver.MarshalAndPushContent(ctx, targetRef, image, ocispec.MediaTypeImageConfig)
	if err != nil {
		return "", fmt.Errorf("unable to update image config: %w", err)
	}
	log.WithField("digest", newDesc.Digest).Debug("image config updated")

	// update the manifest config to point to our new image config's digest
	manifest.Config = newDesc

	///////////
	// Push the image manifest
	///////////
	newDesc, err = resolver.MarshalAndPushContent(ctx, targetRef, manifest, ocispec.MediaTypeImageManifest)
	if err != nil {
		return "", fmt.Errorf("unable to update image manifest: %w", err)
	}
	log.WithField("digest", newDesc.Digest).Debug("manifest updated")

	return newDesc.Digest, nil
}

func (resolver Resolver) GetManifestAndConfigImage(ctx context.Context, ref string, desc ocispec.Descriptor, goarch string) (ocispec.Manifest, ocispec.Image, error) {
	fetcher, err := resolver.Fetcher(ctx, ref)
	if err != nil {
		return ocispec.Manifest{}, ocispec.Image{},
			fmt.Errorf("unable to create fetcher for base image ref %q: %w", ref, err)
	}

	plat := platforms.OnlyStrict(ocispec.Platform{Architecture: goarch, OS: "linux"})
	manifest, err := images.Manifest(ctx, &ProviderWrapper{Fetcher: fetcher}, desc, plat)
	if err != nil {
		return ocispec.Manifest{}, ocispec.Image{},
			fmt.Errorf("unable to fetch manifest of base image ref %q, goarch %q: %w",
				ref, goarch, err)
	}
	log.WithField("descriptor", desc).WithField("ref", ref).Debugf("fetched manifest")

	var image ocispec.Image
	err = FetchAndJSONDecode(ctx, fetcher.Fetch, manifest.Config, &image)
	if err != nil {
		return ocispec.Manifest{}, ocispec.Image{},
			fmt.Errorf("unable to fetch and decode image config: %w", err)
	}

	return manifest, image, nil
}

func AppendLayers(ctx context.Context, store content.Store, desc ocispec.Descriptor, layers []ocispec.Descriptor) (ocispec.Descriptor, ocispec.Descriptor, error) {
	manifest, err := ImageManifestFromProvider(ctx, store, desc)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image manifest (%v) in store: %w", desc, err)
	}

	imageConfig, err := ImageConfigFromProvider(ctx, store, manifest.Config)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, fmt.Errorf("no image config (%v) in store: %w", manifest.Config, err)
	}

    // Get all of the digests of the layers to append to add to the diffids
    // in the image config
	digests := make([]digest.Digest, 0, len(layers))
	for _, layer := range layers {
		digests = append(digests, layer.Digest)
	}

	// Update image with base image reference
	if refName, ok := desc.Annotations[ocispec.AnnotationRefName]; ok {
		refTy, err := dreference.ParseNamed(refName)
		if err != nil {
			return ocispec.Descriptor{}, ocispec.Descriptor{}, err
		}

		ref := refTy.Name()

		for idx, layer := range manifest.Layers {
			if layer.Annotations == nil {
				layer.Annotations = make(map[string]string)
			}

			if _, ok := layer.Annotations[AnnotationBaseImageName]; !ok {
				layer.Annotations[AnnotationBaseImageName] = ref
			}

			if _, ok := layer.Annotations[AnnotationBaseImageDigest]; !ok {
				layer.Annotations[AnnotationBaseImageDigest] = desc.Digest.String()
			}

			manifest.Layers[idx] = layer
		}
	}

    manifest.MediaType = desc.MediaType
	// Append after we add the base image labels
    manifest.Layers = append(manifest.Layers, layers...)
	imageConfig.RootFS.DiffIDs = append(imageConfig.RootFS.DiffIDs, digests...)


	newConfig, err := IngestorJSONEncode(ctx, store, manifest.Config.MediaType, imageConfig)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	manifest.Config = newConfig

	newManifest, err := IngestorJSONEncode(ctx, store, desc.MediaType, manifest)
	if err != nil {
		return ocispec.Descriptor{}, ocispec.Descriptor{}, err
	}

	return newManifest, newConfig, nil
}
