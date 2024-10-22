package ociutil

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// MultiProvider will read from the first provider that can read the requested
// descriptor.
func MultiProvider(providers ...content.Provider) content.Provider {
	return &multiProvider{
		Providers: providers,
	}
}

type multiProvider struct {
	Providers []content.Provider
}

func (mp *multiProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	for _, provider := range mp.Providers {
		ra, err := provider.ReaderAt(ctx, desc)
		if errdefs.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}

		return ra, nil
	}

	return nil, errdefs.ErrNotFound
}
