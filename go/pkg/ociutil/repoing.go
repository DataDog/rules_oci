package ociutil

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	retry "github.com/sethvargo/go-retry"
)

var (
	_ remotes.Resolver   = &extResolver{}
	_ RepositoryIngester = &dockerRegPusher{}
	_ content.Ingester   = &dockerRegPusher{}
)

type RepositoryIngester interface {
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

func (d *dockerRegPusher) Mount(ctx context.Context, from string, digest digest.Digest) error {
	var attempt uint64 = 0
	err := retry.Do(
		ctx,
		retryBackoffStrategy,
		func(ctx context.Context) error {
			wrapErr := func(attempt uint64, msg string, err error) error {
				fmt.Fprintf(
					os.Stderr,
					"failed attempt %d/%d: %s: %v\n",
					attempt,
					retryMaxAttempts,
					msg,
					err,
				)
				return fmt.Errorf("%s: %w", msg, err)
			}

			attempt++

			c := &http.Client{Timeout: 60 * time.Second}

			// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#mounting-a-blob-from-another-repository
			url := fmt.Sprintf(
				"https://%s/v2/%s/blobs/uploads/?mount=%s&from=%s",
				d.registryName,
				d.repo,
				digest.String(),
				from,
			)
			r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
			if err != nil {
				msg := fmt.Sprintf("failed to create request to %q", url)
				return wrapErr(attempt, msg, err)
			}

			if d.registry.Authorizer != nil {
				err = d.registry.Authorizer.Authorize(ctx, r)
				if err != nil {
					msg := fmt.Sprintf("failed to authorize request to %q", url)
					return wrapErr(attempt, msg, err)
				}
			}

			resp, err := c.Do(r)
			if err != nil {
				msg := fmt.Sprintf("failed to create request to %q", url)
				return wrapErr(attempt, msg, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusCreated {
				return nil
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				msg := fmt.Sprintf(
					"invalid status code received from %q (%d): unable to read body",
					url,
					resp.StatusCode,
				)
				return wrapErr(attempt, msg, err)
			}

			msg := fmt.Sprintf(
				"invalid status code received from %q (%d)",
				url,
				resp.StatusCode,
			)
			err = errors.New(string(body))
			return wrapErr(attempt, msg, err)
		},
	)
	if err != nil {
		return err
	}
	return nil
}
