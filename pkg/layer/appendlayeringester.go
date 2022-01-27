package layer

import (
	"context"
	"fmt"
	"os"
    "sync"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
)

var (
    ErrNotSupportedMediaType = fmt.Errorf("not supported media type for ingester")

	_ content.Ingester = &appendIngester{}
	_ content.Writer   = &fileWriter{}
)

// NewAppendIngester creates a containerd/content.Ingestor that will write
// out the modified files to a predefined paths. The descriptor WriterOpt is
// required with an accurate MediaType.
//
// This is useful in a Bazel context as we need to predeclare where files will
// be written to.
func NewAppendIngester(manifestPath, configPath string) content.Ingester {
	return &appendIngester{
		manifestPath: manifestPath,
		configPath:   configPath,
	}
}

type appendIngester struct {
	manifestPath string
	configPath   string

    mx sync.Mutex
}

func (ing *appendIngester) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	var options content.WriterOpts

	for _, o := range opts {
		if err := o(&options); err != nil {
			return nil, err
		}
	}

    ing.mx.Lock()
    defer ing.mx.Unlock()

	path := ""
	if images.IsManifestType(options.Desc.MediaType) {
		if ing.manifestPath != "" {
            return nil, fmt.Errorf("%w: already seen manifest", ErrNotSupportedMediaType)
        }

        path = ing.manifestPath
    } else if images.IsConfigType(options.Desc.MediaType) {
		if ing.configPath != "" {
            return nil, fmt.Errorf("%w: already seen config", ErrNotSupportedMediaType)
        }

        path = ing.configPath
	} else {
        return nil, fmt.Errorf("%w: %v", ErrNotSupportedMediaType, options.Desc.MediaType)
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
	return fmt.Errorf("append layer file writer: %w", errdefs.ErrNotImplemented)
}
