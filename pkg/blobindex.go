package ociutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
    //	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
	_ content.Provider = &BlobIndex{}
	_ content.ReaderAt = &fileWithSize{}
)

/*
func appendBlobIndexHandler(bi *BlobIndex, handler images.HandlerFunc) images.HandlerFunc {
    return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
        bi[]
    }
}

func BlobIndexFromHandler(ctx context.Context, provider content.Provider, root ocispec.Descriptor) (*BlobIndex, error) {
    err := images.Walk(ctx, images.ChildrenHandler(provider), root)
    if err != nil {
        return nil, err 
    }

}
*/

func MergeProviders(providers ...content.Provider) (*BlobIndex, error) {
    bi := &BlobIndex{}

    for _, p := range providers {
        if b, ok := p.(*BlobIndex); ok {
            bi.Merge(b)
        } else {
            return nil, fmt.Errorf("only supports BlobIndex")
        }
    }

    return bi, nil
}


// LoadBlobIndex loads an JSON file that is an index of all of the blobs in a
// OCI image index.
//
// This is useful in Bazel so you don't need to repackage all of the entries in
// an OCI Layout.
func LoadBlobIndex(path string) (content.Provider, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open index: %w", err)
	}

	var idx BlobIndex

	err = json.NewDecoder(f).Decode(&idx)
	if err != nil {
		return nil, err
	}

	return &idx, nil
}

// Rel creates a new index with all paths relative to the provided path.
//
// This is used in Bazel when using path vs short_path.
func (bi *BlobIndex) Rel(rel string) (*BlobIndex, error) {
	clone := bi.Clone()

	for dgst, path := range clone.Blobs {
		newPath, err := filepath.Rel(rel, path)
		if err != nil {
			return nil, err
		}

		clone.Blobs[dgst] = newPath
	}

	return clone, nil
}

func (bi *BlobIndex) Merge(bim *BlobIndex) {
    if bi.Blobs == nil {
        bi.Blobs = make(map[digest.Digest]string)
    }

    for k, v := range bim.Blobs {
        bi.Blobs[k] = v
    }

}

func (bi *BlobIndex) Clone() *BlobIndex {
	newbi := &BlobIndex{
		Blobs: make(map[digest.Digest]string),
	}

	for dgst, path := range bi.Blobs {
		newbi.Blobs[dgst] = path
	}

	return newbi
}

// WriteTo writes the index to a stream.
func (bi *BlobIndex) WriteTo(writer io.Writer) error {
	return json.NewEncoder(writer).Encode(bi)
}

// WriteToFile writes the index to a file.
func (bi *BlobIndex) WriteToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return bi.WriteTo(f)
}

// BlobIndex is a mapping from digest to a filepath
type BlobIndex struct {
	// TODO(griffin): add support for top level index
	// Index digest.Digest

	Blobs map[digest.Digest]string
}

func (bi *BlobIndex) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	path, ok := bi.Blobs[desc.Digest]
	if !ok {
		return nil, errdefs.ErrNotFound
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &fileWithSize{
		File: f,
		size: fi.Size(),
	}, nil
}

type fileWithSize struct {
	*os.File
	size int64
}

func (f *fileWithSize) Size() int64 {
	return f.size
}
