package ociutil

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	_ remotes.Resolver   = &extResolver{}
	_ RepositoryIngester = &dockerRegPusher{}
	_ content.Ingester   = &dockerRegPusher{}
)

type RepositoryIngester interface {
	Contains(ctx context.Context, dgst digest.Digest) error
	Mount(ctx context.Context, from string, dgst digest.Digest) error
}

func ExtendedResolver(resolver remotes.Resolver, hosts docker.RegistryHosts) remotes.Resolver {
	return &extResolver{resolver, hosts}
}

type extResolver struct {
	resolver remotes.Resolver
	hosts    docker.RegistryHosts
}

func (r *extResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	pusher, err := r.resolver.Pusher(ctx, ref)
	if err != nil {
		return nil, err
	}

	regName, err := RefToRegistryName(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	repo, err := RefToPath(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}

	matchedRegistries, err := r.hosts(regName)
	if err != nil {
		return nil, fmt.Errorf("failed to find auth: %w", err)
	}

	var registry docker.RegistryHost
	if len(matchedRegistries) > 0 {
		registry = matchedRegistries[0]
	}

	return &dockerRegPusher{
		Pusher:       pusher,
		registryName: regName,
		repo:         repo,
		registry:     registry,
	}, nil
}

func (r *extResolver) Fetcher(ctx context.Context, ref string) (remotes.Fetcher, error) {
	return r.resolver.Fetcher(ctx, ref)
}

func (r *extResolver) Resolve(ctx context.Context, ref string) (string, ocispec.Descriptor, error) {
	return r.resolver.Resolve(ctx, ref)
}

type dockerRegPusher struct {
	remotes.Pusher
	registryName string
	repo         string
	registry     docker.RegistryHost
}

func (p *dockerRegPusher) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	if ing, ok := p.Pusher.(content.Ingester); ok {
		return ing.Writer(ctx, opts...)
	}

	return nil, errdefs.ErrNotImplemented
}

func (p *dockerRegPusher) Contains(ctx context.Context, dgst digest.Digest) error {
	return fmt.Errorf("contains not implemented")
}

func (d *dockerRegPusher) Mount(ctx context.Context, from string, dgst digest.Digest) error {
	hc := &http.Client{Timeout: 60 * time.Second}

	// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#mounting-a-blob-from-another-repository
	mountURL := fmt.Sprintf("https://%s/v2/%s/blobs/uploads/?mount=%s&from=%s", d.registryName, d.repo, dgst.String(), from)
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, mountURL, nil)
	if err != nil {
		return fmt.Errorf("unable to create mount http request: %w", err)
	}

	if d.registry.Authorizer != nil {
		err = d.registry.Authorizer.Authorize(ctx, r)
		if err != nil {
			return fmt.Errorf("couldn't authorize mount request: %w", err)
		}
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

	return nil
}
