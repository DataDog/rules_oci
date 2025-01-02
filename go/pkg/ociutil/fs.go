package ociutil

import (
	"context"
	"io"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/containerd/containerd/content"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type iofsProvider struct {
	FS fs.FS
}

func (bi *iofsProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {

	f, err := bi.FS.Open(descToFilePath("", desc.Digest))
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	return &fsFile{
		File: f,
		size: fi.Size(),
	}, nil
}

type fsFile struct {
	fs.File
	reopen func() (fs.File, error)
	size   int64
	offset int64
	mx     sync.Mutex
}

func (f *fsFile) Size() int64 {
	return f.size
}

func seekForward(f fs.File, offset int64) error {
	if sk, ok := f.(io.Seeker); ok {
		_, err := sk.Seek(offset, io.SeekCurrent)
		return err
	}

	_, err := io.CopyN(io.Discard, f, offset)
	if err != nil {
		return err
	}

	return nil
}

func (f *fsFile) ReadAt(p []byte, off int64) (int, error) {
	f.mx.Lock()
	defer f.mx.Unlock()

	if ra, ok := f.File.(io.ReaderAt); ok {
		return ra.ReadAt(p, off)
	}

	if f.File == nil || f.offset > off {
		f.Close()
		newFile, err := f.reopen()
		if err != nil {
			return 0, err
		}

		f.File = newFile
		f.offset = 0
	}

	if off > f.offset {
		err := seekForward(f, off-f.offset)
		if err != nil {
			return 0, err
		}
	}

	n, err := f.Read(p)
	if err != nil {
		return 0, err
	}
	f.offset += int64(n)

	return n, nil
}

func descToFilePath(root string, dgst digest.Digest) string {
	return filepath.Join(root, "blobs", dgst.Algorithm().String(), dgst.Encoded())
}
