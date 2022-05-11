// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"archive/tar"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
)

var (
	_ content.Ingester = &tarIngestor{}
)

func NewTarIngestor(path string) content.Ingester {
	return &tarIngestor{path: path}
}

type tarIngestor struct {
	path string
	tw   *tar.Writer
	sync.Mutex
	wo content.WriterOpts
}

func (ing *tarIngestor) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	ing.Lock()

	for _, o := range opts {
		if err := o(&ing.wo); err != nil {
			ing.Close()
			return nil, err
		}
	}

	f, err := os.OpenFile(ing.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		ing.Close()
		return nil, err
	}

	ing.tw = tar.NewWriter(f)

	dgst := ing.wo.Desc.Digest
	if err := dgst.Validate(); err != nil {
		ing.Close()
		return nil, fmt.Errorf("tar ingestor: must have digest: %w", err)
	}

	hdr := &tar.Header{
		Name: fmt.Sprintf("blobs/%v/%v", dgst.Algorithm().String(), dgst.Encoded()),
		Mode: 0600,
		Size: ing.wo.Desc.Size,
	}

	err = ing.tw.WriteHeader(hdr)
	if err != nil {
		ing.Close()
		return nil, err
	}

	return ing, nil
}

func (ing *tarIngestor) Close() error {
	ing.tw.Close()
	ing.wo = content.WriterOpts{}
	ing.Unlock()
	return nil
}

func (ing *tarIngestor) Write(p []byte) (int, error) {
	return ing.tw.Write(p)
}

func (ing *tarIngestor) Digest() digest.Digest {
	return digest.Digest("")
}

func (ing *tarIngestor) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	return ing.Close()
}

func (ing *tarIngestor) Status() (content.Status, error) {
	return content.Status{}, fmt.Errorf("tar writer: not implemented")
}

func (ing *tarIngestor) Truncate(size int64) error {
	return fmt.Errorf("tar writer: not implemented")
}
