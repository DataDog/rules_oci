package ociutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/DataDog/rules_oci/pkg/credhelper"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	dref "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

// DefaultResolver returns a resolver with credential helper auth and ocitool
// extensions.
func DefaultResolver() Resolver {
	return newResolver(nil)
}

// ResolverWithHeaders returns a resolver with credential helper auth and ocitool
// extensions.
func ResolverWithHeaders(headers map[string]string) Resolver {
	return newResolver(headers)
}

func newResolver(headers map[string]string) Resolver {
	hdrs := http.Header{}
	for k, v := range headers {
		hdrs.Add(k, v)
	}

	hosts := docker.Registries(
		credhelper.RegistryHostsFromDockerConfig(),
		// Support for Docker Hub
		docker.ConfigureDefaultRegistries(),
	)

	return Resolver{
		Resolver: ExtendedResolver(docker.NewResolver(docker.ResolverOptions{
			Hosts:   hosts,
			Headers: hdrs,
		}), hosts),
	}
}

type Resolver struct {
	remotes.Resolver
}

// NamedRef will parse the ref and return a Named instance of it.
func NamedRef(ref string) (dref.Named, error) {
	refr, err := dref.Parse(ref)
	if err != nil {
		return nil, err
	}

	n, ok := refr.(dref.Named)
	if !ok {
		return nil, fmt.Errorf("not a named reference")
	}

	return n, nil
}

// RefToPath will parse the ref and just its path
func RefToPath(ref string) (string, error) {
	n, err := NamedRef(ref)
	if err != nil {
		return "", err
	}

	return dref.Path(n), nil
}

// RefToRegistryName will return a hostname of a registry given a reference string.
func RefToRegistryName(ref string) (string, error) {
	n, err := NamedRef(ref)
	if err != nil {
		return "", err
	}

	return dref.Domain(n), nil
}

// PushBlob pushes a singluar blob to a registry.
func (resolver Resolver) PushBlob(ctx context.Context, path, ref, mediaType string) (ocispec.Descriptor, error) {
	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer f.Close()

	fileInfo, err := os.Stat(path)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	dig, err := digest.FromReader(f)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      fileInfo.Size(),
		Digest:    dig,
	}

	writer, err := pusher.Push(ctx, desc)
	if errdefs.IsAlreadyExists(err) {
		return desc, nil
	}

	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer writer.Close()

	_, err = f.Seek(0, 0)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	err = content.Copy(ctx, writer, f, fileInfo.Size(), dig)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return desc, err
}

// PushImageIndexShallow pushes a new image index to a repository without
// pulling all of the dependent descriptors, aka it doesn't need to pull any of
// the dependent images.
//
// XXX Currently there is a major limitation that you can only push an index
// who's dependencies all coexist within the same repository. To support
// cross-repository shallow pushes we would need to mount blobs to this
// repository. `containerd` has some nice facilities to walk descriptors.
// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#mounting-a-blob-from-another-repository
func (resolver Resolver) PushImageIndexShallow(ctx context.Context, idx ocispec.Index, ref string) (ocispec.Descriptor, error) {

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Size:      int64(len(data)),
		Digest:    digest.SHA256.FromBytes(data),
	}

	writer, err := pusher.Push(ctx, desc)
	if errdefs.IsAlreadyExists(err) {
		return ocispec.Descriptor{}, err
	}
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer writer.Close()

	reader := bytes.NewBuffer(data)
	_, err = content.CopyReader(writer, reader)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	err = writer.Commit(ctx, desc.Size, desc.Digest)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return desc, nil
}

func (resolver Resolver) MarshalAndPushContent(ctx context.Context, ref string, content interface{}, mediaType string) (ocispec.Descriptor, error) {
	contents, err := json.Marshal(content)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("unable to marshal JSON: %w", err)
	}
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      int64(len(contents)),
		Digest:    digest.SHA256.FromBytes(contents),
	}

	pusher, err := resolver.Pusher(ctx, ref)
	if err != nil {
		return desc, fmt.Errorf("unable to build pusher: %w", err)
	}
	w, err := pusher.Push(ctx, desc)
	if err != nil {
		if errors.Is(err, errdefs.ErrAlreadyExists) {
			return desc, nil
		}
		return desc, fmt.Errorf("unable to build pusher writer: %w", err)
	}
	_, err = w.Write(contents)
	if err != nil {
		return desc, fmt.Errorf("unable to push image: %w", err)
	}
	err = w.Close()
	if err != nil {
		return desc, fmt.Errorf("unable to close writer: %w", err)
	}
	log.Debug("contents pushed")

	err = w.Commit(ctx, desc.Size, desc.Digest)
	if err != nil {
		return desc, fmt.Errorf("unable to commit: %w", err)
	}

	log.Debug("contents committed")

	return desc, nil
}

// CopyContent copies a descriptor from a provider to an ingestor interfaces
// provider by "containerd/content". Useful when you want to copy between
// layouts or when pulling an image via oras.ProviderWrapper
func CopyContent(ctx context.Context, from content.Provider, to content.Ingester, desc ocispec.Descriptor) error {
	logCtx := log.WithField("digest", desc.Digest).WithField("desc", desc)

	// If we're talking to an OCI registry we can take some shortcuts by
	// checking for the existence of the blob.
	if reg, ok := to.(RepositoryIngester); ok {

		// First check if it exists in the repository we're pushing to (always fails, because the
		// only implementation of RepositoryIngester (ociutil.dockerRegPusher) doesn't implement
		// Contains).
		err := reg.Contains(ctx, desc.Digest)
		if err == nil {
			logCtx.Debugf("skipped copy, already exist")
			return nil
		}
		logCtx.WithError(err).Debug("blob doesn't exist")

		// If we know which repo the blob is from (see layers.AppendLayers for how
		// AnnotationBaseImageName is set in layer descriptors; otherwise, the presence of this
		// annotation may not mean that that a descriptor is _from_ an image, rather it means it
		// _has_ that image as a base), then lets try to mount it into the new repo. This code
		// should probably check that RefToDomain(ref) matches the registry represented by `to`
		// before attempting the Mount call, or the Mount call should do this.
		//
		// TODO: should also allow ocispec.AnnotationRefName
		if ref, ok := desc.Annotations[AnnotationBaseImageName]; ok {
			repo, err := RefToPath(ref)
			if err != nil {
				return fmt.Errorf("failed to mount blob: %w", err)
			}

			err = reg.Mount(ctx, repo, desc.Digest)
			if err == nil {
				logCtx.Debugf("skipped copy, mounted blob from %q", repo)
				return nil
			}
			logCtx.WithError(err).Debug("couldn't mount blob")
		}
	}

	reader, err := from.ReaderAt(ctx, desc)
	if err != nil {
		return fmt.Errorf("failed to create reader from ingestor: %w", err)
	}

	ref := desc.Digest.String()
	if refAnno, ok := desc.Annotations[ocispec.AnnotationRefName]; ok {
		ref = refAnno
	}

	cw, err := to.Writer(ctx, content.WithRef(ref), content.WithDescriptor(desc))
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to write content to ingestor: %w", err)
	}

	err = content.Copy(ctx, cw, content.NewReader(reader), desc.Size, desc.Digest)
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to copy content from provider to ingestor: %w", err)
	}

	return nil
}
