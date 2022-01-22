package ociutil

import (
	"context"
	"fmt"
	"os"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
)

var (
	_ content.Ingester = &appendLayerIngester{}
	_ content.Writer   = &fileWriter{}
)

// NewAppendLayerIngestor creates a containerd/content.Ingestor that will write
// out the modified files to a predefined paths. The descriptor WriterOpt is
// required with an accurate MediaType.
//
// This is useful in a Bazel context as we need to predeclare where files will
// be written to.
func NewAppendLayerIngestor(manifestPath, configPath string) content.Ingester {
	return &appendLayerIngester{
		manifestPath: manifestPath,
		configPath:   configPath,
	}
}

type appendLayerIngester struct {
	manifestPath string
	configPath   string
}

func (ing *appendLayerIngester) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	var options content.WriterOpts

	for _, o := range opts {
		if err := o(&options); err != nil {
			return nil, err
		}
	}

	path := ""
	if images.IsManifestType(options.Desc.MediaType) {
		path = ing.manifestPath
	} else if images.IsConfigType(options.Desc.MediaType) {
		path = ing.configPath
	} else {
		return nil, fmt.Errorf("not supported content type for writer")
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}

	return &fileWriter{
		f: f,
	}, nil
}

type fileWriter struct {
	f   *os.File
	off int
}

func (ing *fileWriter) Close() error {
	return ing.f.Close()
}

func (ing *fileWriter) Write(p []byte) (int, error) {
	n, err := ing.f.Write(p)
	if err != nil {
		return 0, err
	}

	ing.off += n

	return n, err
}

func (ing *fileWriter) Digest() digest.Digest {
	return digest.Digest("")
}

func (ing *fileWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	return ing.Close()
}

func (ing *fileWriter) Status() (content.Status, error) {
	return content.Status{
		Offset: int64(ing.off),
	}, nil
}

func (ing *fileWriter) Truncate(size int64) error {
	return fmt.Errorf("file writer: %w", errdefs.ErrNotImplemented)
}
