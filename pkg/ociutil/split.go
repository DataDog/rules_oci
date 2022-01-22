package ociutil

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	_ content.Store = &splitStore{}
)

// SplitStore implementents content.Store, where reads are from a different
// store than writes.
func SplitStore(ing content.Ingester, prov content.Provider) content.Store {
	return &splitStore{
		ingester: ing,
		provider: prov,
	}
}

type splitStore struct {
	ingester content.Ingester
	provider content.Provider
}

func (s *splitStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	return s.provider.ReaderAt(ctx, desc)
}

func (s *splitStore) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	return s.ingester.Writer(ctx, opts...)
}

func (s *splitStore) Status(ctx context.Context, ref string) (content.Status, error) {
	if im, ok := s.ingester.(content.IngestManager); ok {
		return im.Status(ctx, ref)
	}

	return content.Status{}, errdefs.ErrNotImplemented
}

func (s *splitStore) ListStatuses(ctx context.Context, filters ...string) ([]content.Status, error) {
	if im, ok := s.ingester.(content.IngestManager); ok {
		return im.ListStatuses(ctx, filters...)
	}

	return nil, errdefs.ErrNotImplemented
}

func (s *splitStore) Abort(ctx context.Context, ref string) error {
	if im, ok := s.ingester.(content.IngestManager); ok {
		return im.Abort(ctx, ref)
	}

	return errdefs.ErrNotImplemented
}

func (s *splitStore) Info(ctx context.Context, dgst digest.Digest) (content.Info, error) {
	if man, ok := s.provider.(content.Manager); ok {
		return man.Info(ctx, dgst)
	}

	return content.Info{}, errdefs.ErrNotImplemented
}

func (s *splitStore) Update(ctx context.Context, info content.Info, fieldpaths ...string) (content.Info, error) {
	if man, ok := s.provider.(content.Manager); ok {
		return man.Update(ctx, info, fieldpaths...)
	}

	return content.Info{}, errdefs.ErrNotImplemented
}

func (s *splitStore) Walk(ctx context.Context, fn content.WalkFunc, filters ...string) error {
	if man, ok := s.provider.(content.Manager); ok {
		return man.Walk(ctx, fn, filters...)
	}

	return errdefs.ErrNotImplemented
}

func (s *splitStore) Delete(ctx context.Context, dgst digest.Digest) error {
	if man, ok := s.provider.(content.Manager); ok {
		return man.Delete(ctx, dgst)
	}

	return errdefs.ErrNotImplemented
}
