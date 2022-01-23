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
	"github.com/opencontainers/go-digest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

var (
    ErrNotBlobIndex = fmt.Errorf("provider not a blob index")

	_ content.Provider = &Index{}
	_ content.ReaderAt = &fileWithSize{}
)

func MergeIndex(providers ...content.Provider) (*Index, error) {
	bi := &Index{}

	for _, p := range providers {
		if b, ok := p.(*Index); ok {
			bi.Merge(b)
		} else {
			return nil, ErrNotBlobIndex
		}
	}

	return bi, nil
}

// LoadBlobIndex loads an JSON file that is an index of all of the blobs in a
// OCI image index.
//
// This is useful in Bazel so you don't need to repackage all of the entries in
// an OCI Layout.
func LoadIndexFromFile(path string) (content.Provider, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open index: %w", err)
	}

	var idx Index

	err = json.NewDecoder(f).Decode(&idx)
	if err != nil {
		return nil, err
	}

	return &idx, nil
}

// Rel creates a new index with all paths relative to the provided path.
//
// This is used in Bazel when using path vs short_path.
func (bi *Index) Rel(rel string) (*Index, error) {
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

func (bi *Index) Merge(bim *Index) {
	if bi.Blobs == nil {
		bi.Blobs = make(map[digest.Digest]string)
	}

	for k, v := range bim.Blobs {
		bi.Blobs[k] = v
	}

}

func (bi *Index) Clone() *Index {
	newbi := &Index{
		Blobs: make(map[digest.Digest]string),
	}

	for dgst, path := range bi.Blobs {
		newbi.Blobs[dgst] = path
	}

	return newbi
}

// WriteTo writes the index to a stream.
func (bi *Index) WriteTo(writer io.Writer) error {
	return json.NewEncoder(writer).Encode(bi)
}

// WriteToFile writes the index to a file.
func (bi *Index) WriteToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return bi.WriteTo(f)
}

// BlobIndex is a mapping from digest to a filepath
type Index struct {
	// TODO(griffin): add support for top level index
	// Index digest.Digest

	Blobs map[digest.Digest]string
}

func (bi *Index) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
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
