package ociutil

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/DataDog/zstd"
	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// GetLayerDiffID will return the diff ID of a given layer descriptor to be used within an
// image config. If a layer is uncompressed, the diff ID is simply the digest but if the
// layer is compressed, we must uncompress the file and acquire the digest.
func GetLayerDiffID(ctx context.Context, store content.Store, desc ocispec.Descriptor) (digest.Digest, error) {
	if desc.MediaType != ocispec.MediaTypeImageLayerGzip && desc.MediaType != ocispec.MediaTypeImageLayerZstd {
		return desc.Digest, nil
	}

	r, err := store.ReaderAt(ctx, desc)
	if err != nil {
		return "", fmt.Errorf("failed to get reader for layer: %w", err)
	}
	defer r.Close()

	var cr io.Reader
	switch desc.MediaType {
	case ocispec.MediaTypeImageLayerGzip:
		cr, err = gzip.NewReader(&readerAtReader{ReaderAt: r})
		if err != nil {
			return "", fmt.Errorf("failed to get gzip reader for layer: %w", err)
		}
	case ocispec.MediaTypeImageLayerZstd:
		cr = zstd.NewReader(&readerAtReader{ReaderAt: r})
	}

	return digest.SHA256.FromReader(cr)
}

type readerAtReader struct {
	content.ReaderAt
	offset int64
}

func (r *readerAtReader) Read(b []byte) (int, error) {
	n, err := r.ReadAt(b, r.offset)
	r.offset += int64(n)
	return n, err
}
