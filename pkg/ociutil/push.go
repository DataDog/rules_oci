package ociutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	dref "github.com/containerd/containerd/reference/docker"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

// NewDDRegistryResolver returns a general purpose resolver for use with
// dd-registry.
//
// TODO(griffin): Add built-in Vault auth
func NewDDRegistryResolver() Resolver {
	return Resolver{Resolver: ExtendedResolver(docker.NewResolver(docker.ResolverOptions{}))}
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

// RefToHostname will return a hostname of a registry given a reference string.
func RefToHostname(ref string) (string, error) {
	n, err := NamedRef(ref)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%v", dref.Domain(n)), nil
}

func (resolver Resolver) MountBlobs(ctx context.Context, baseDesc ocispec.Descriptor, baseRef, targetRef, goarch string) error {
	baseHost, err := RefToHostname(baseRef)
	if err != nil {
		return fmt.Errorf("unable to parse base ref: %w", err)
	}
	targetHost, err := RefToHostname(targetRef)
	if err != nil {
		return fmt.Errorf("unable to parse base ref: %w", err)
	}
	// we're copying from one repo to another, not one registry to another
	if baseHost != targetHost {
		return fmt.Errorf("base ref host %q must match target %q", baseHost, targetHost)
	}

	baseName, err := RefToPath(baseRef)
	if err != nil {
		return fmt.Errorf("unable to parse target ref: %w", err)
	}
	targetName, err := RefToPath(targetRef)
	if err != nil {
		return fmt.Errorf("unable to parse target ref: %w", err)
	}

	fetcher, err := resolver.Fetcher(ctx, baseRef)
	if err != nil {
		return fmt.Errorf("unable to fetch base ref: %w", err)
	}

	plat := platforms.OnlyStrict(ocispec.Platform{Architecture: goarch, OS: "linux"})
	manifest, err := images.Manifest(ctx, &ProviderWrapper{Fetcher: fetcher}, baseDesc, plat)
	if err != nil {
		return fmt.Errorf("unable to fetch base manifest: %w", err)
	}

	hc := &http.Client{Timeout: 60 * time.Second}
	for _, layer := range manifest.Layers {
		// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#mounting-a-blob-from-another-repository
		mountURL := fmt.Sprintf("%s/v2/%s/blobs/uploads/?mount=%s&from=%s", baseHost,
			targetName, layer.Digest, baseName)
		r, err := http.NewRequestWithContext(ctx, http.MethodPost, mountURL, nil)
		if err != nil {
			return fmt.Errorf("unable to create mount http request: %w", err)
		}

		resp, err := hc.Do(r)
		if err != nil {
			return fmt.Errorf("unable to make mount request: %w", err)
		}
		err = func() error {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusCreated {
				return nil
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("invalid status code from %q: %d. unable to read body: %w",
					mountURL, resp.StatusCode, err)
			}
			return fmt.Errorf("invalid status code from %q (%d): %s",
				mountURL, resp.StatusCode, string(body))
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func (resolver Resolver) ConvertDebsAndPushBlobs(ctx context.Context, ref string, urls []string) ([]ocispec.Descriptor, []digest.Digest, error) {
	dir, err := ioutil.TempDir("", "*-dl-and-push-blobs")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// for each file, download it and create a layer
	layers := make([]ocispec.Descriptor, len(urls))
	digests := make([]digest.Digest, len(urls))

	for i, url := range urls {
		file, err := os.CreateTemp(dir, "*.tar")
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create file: %w", err)
		}

		filePath := file.Name()

		resp, err := http.Get(url)
		if err != nil {
			file.Close()
			return nil, nil, err
		}

		log.WithField("url", url).Debug("downloaded deb")

		err = DebToLayer(resp.Body, file)
		resp.Body.Close()
		file.Close()
		if err != nil {
			return nil, nil, err
		}

		layer, err := resolver.PushBlob(ctx, filePath, ref, ocispec.MediaTypeImageLayer)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to push file %q: %w", filePath, err)
		}
		log.WithField("digest", layer.Digest).
			Debug("pushed layer contents to registry")

		layers[i] = layer
		digests[i] = layer.Digest
	}

	return layers, digests, nil
}

func (resolver Resolver) DownloadAndPushBlobs(ctx context.Context, bucket, region string, keys []string, ref string) ([]ocispec.Descriptor, []digest.Digest, error) {
	dir, err := ioutil.TempDir("", "*-dl-and-push-blobs")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// for each file, download it and create a layer
	layers := make([]ocispec.Descriptor, len(keys))
	digests := make([]digest.Digest, len(keys))

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to start a new AWS SDK session: %w", err)
	}
	dl := s3manager.NewDownloader(sess)

	for i, key := range keys {
		filePath := path.Join(dir, path.Base(key))
		file, err := os.Create(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create file %q: %w", filePath, err)
		}

		size, err := dl.DownloadWithContext(ctx, file, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return nil, nil, err
		}

		log.WithFields(log.Fields{
			"bucket": bucket,
			"key":    key,
			"path":   filePath,
			"size":   size,
		}).Debug("downloaded file")

		layer, err := resolver.PushBlob(ctx, filePath, ref, ocispec.MediaTypeImageLayer)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to push file %q: %w", filePath, err)
		}
		log.WithField("digest", layer.Digest).
			Debug("pushed layer contents to registry")

		layers[i] = layer
		digests[i] = layer.Digest
	}

	return layers, digests, nil
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

		// First check if it exists in the repository we're pushing to
		err := reg.Contains(ctx, desc.Digest)
		if err == nil {
			logCtx.Debugf("skipped copy, already exist")
			return nil
		}
		logCtx.WithError(err).Debug("blob doesn't exist")

		// If we know where the blob is from, then lets try to mount it
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
		return err
	}

	ref := desc.Digest.String()
	if refAnno, ok := desc.Annotations[ocispec.AnnotationRefName]; ok {
		ref = refAnno
	}

	cw, err := to.Writer(ctx, content.WithRef(ref), content.WithDescriptor(desc))
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		return nil
	}
	if err != nil {
		return err
	}

	err = content.Copy(ctx, cw, content.NewReader(reader), desc.Size, desc.Digest)
	if errors.Is(err, errdefs.ErrAlreadyExists) {
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}
