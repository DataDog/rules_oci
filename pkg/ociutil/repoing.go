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

func ExtendedResolver(resolver remotes.Resolver) remotes.Resolver {
	return &extResolver{resolver}
}

type extResolver struct {
	resolver remotes.Resolver
}

func (r *extResolver) Pusher(ctx context.Context, ref string) (remotes.Pusher, error) {
	pusher, err := r.resolver.Pusher(ctx, ref)
	if err != nil {
		return nil, err
	}

	host, err := RefToHostname(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}
	repo, err := RefToPath(ref)
	if err != nil {
		return nil, fmt.Errorf("unable to parse ref: %w", err)
	}

	return &dockerRegPusher{
		Pusher: pusher,
		host:   host,
		repo:   repo,
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
	host string
	repo string
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
	mountURL := fmt.Sprintf("%s/v2/%s/blobs/uploads/?mount=%s&from=%s", d.host, d.repo, dgst.String(), from)
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

	return nil
}
