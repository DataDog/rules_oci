package ociutil

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// GetLayerDiffID will return the diff ID of a given layer descriptor to be used within an
// image config. If a layer is uncompressed, the diff ID is simply the digest but if the
// layer is compressed, we must uncompress the file and acquire the digest.
func GetLayerDiffID(ctx context.Context, store content.Store, desc ocispec.Descriptor) (digest.Digest, error) {
	switch desc.MediaType {
	case ocispec.MediaTypeImageLayerGzip:
		r, err := store.ReaderAt(ctx, desc)
		if err != nil {
			return "", fmt.Errorf("failed to get reader for layer: %w", err)
		}
		defer r.Close()

		gr, err := gzip.NewReader(&readerAtReader{ReaderAt: r})
		if err != nil {
			return "", fmt.Errorf("failed to get gzip reader for layer: %w", err)
		}

		return digest.SHA256.FromReader(gr)
	case ocispec.MediaTypeImageLayerZstd:
		r, err := store.ReaderAt(ctx, desc)
		if err != nil {
			return "", fmt.Errorf("failed to get reader for layer: %w", err)
		}
		defer r.Close()

		zr, err := zstd.NewReader(&readerAtReader{ReaderAt: r})
        if err != nil {
		    return "", err
	    }
		return digest.SHA256.FromReader(zr)
	default:
		return desc.Digest, nil
	}
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
