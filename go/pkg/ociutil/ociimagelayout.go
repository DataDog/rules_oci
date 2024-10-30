package ociutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
)

const BlobsFolderName = "blobs"
const OciImageIndexMediaType = "application/vnd.oci.image.index.v1+json"
const OciLayoutFileName = "oci-layout"
const OciLayoutFileContent = `{
    "imageLayoutVersion": "1.0.0"
}`
const OciIndexFileName = "index.json"
const ContentFileMode = 0755

// OciImageLayoutIngester implements functionality to write data to an OCI
// Image Layout directory (https://github.com/opencontainers/image-spec/blob/main/image-layout.md)
type OciImageLayoutIngester struct {
	// The path of the directory containing the OCI Image Layout.
	Path string
}

func NewOciImageLayoutIngester(path string) (OciImageLayoutIngester, error) {
	if err := os.MkdirAll(path, ContentFileMode); err != nil {
		return OciImageLayoutIngester{}, fmt.Errorf("error creating directory for OciImageLayoutIngester: %v, Err: %w", path, err)
	}
	return OciImageLayoutIngester{Path: path}, nil
}

// writer returns a Writer object that will write one entity to the OCI Image Layout.
// Examples are OCI Image Index, an OCI Image Manifest, an OCI Image Config,
// and OCI image TAR/GZIP files.
func (ing *OciImageLayoutIngester) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	// Initialize the writer options (for those unfamiliar with this pattern, it's known as the
	// "functional options pattern").
	var wOpts content.WriterOpts
	for _, o := range opts {
		if err := o(&wOpts); err != nil {
			return nil, fmt.Errorf("unable to apply WriterOpt to WriterOpts. Err: %w", err)
		}
	}
	status := content.Status{
		Ref:      wOpts.Ref,
		Offset:   0,
		Total:    wOpts.Desc.Size,
		Expected: wOpts.Desc.Digest,
	}
	return &OciImageLayoutWriter{Path: ing.Path, Opts: wOpts, Stat: status}, nil
}

type OciImageLayoutWriter struct {
	Path string
	Opts content.WriterOpts
	Dig  digest.Digest
	Stat content.Status
}

// Writes bytes to a file, using the provided write flags.
func writeFile(filePath string, writeFlags int, b []byte) error {
	f, err := os.OpenFile(filePath, writeFlags, ContentFileMode)
	if err != nil {
		return fmt.Errorf("error opening file for write: %v, Err: %w", filePath, err)
	}
	defer f.Close()

	if _, err = f.Write(b); err != nil {
		return fmt.Errorf("error writing file: %v, Err: %w", filePath, err)
	}
	return nil
}

func (w *OciImageLayoutWriter) Write(b []byte) (n int, err error) {
	firstWrite := w.Stat.StartedAt.IsZero()
	if firstWrite {
		w.Stat.StartedAt = time.Now()
	}

	// A function to get the OS flags used to create a writeable file.
	getWriteFlags := func(filePath string) (int, error) {
		_, err := os.Stat(filePath)
		switch {
		case firstWrite && err == nil:
			// The file exists and it's first write; Truncate it.
			return os.O_WRONLY | os.O_TRUNC, nil
		case err == nil:
			// The file exists and it's not first write; append to it.
			return os.O_WRONLY | os.O_APPEND, nil
		case errors.Is(err, os.ErrNotExist):
			// The file doesn't exist. Create it.
			return os.O_WRONLY | os.O_CREATE, nil
		default:
			// Something went wrong!
			return 0, err
		}
	}

	var filePath string
	if w.Opts.Desc.MediaType == OciImageIndexMediaType {
		// This is an OCI Image Index. It gets written to the top level index.json file.
		// Write the oci-layout file (a simple canned file required by the standard).
		layoutFile := path.Join(w.Path, OciLayoutFileName)
		// It's possible (but unlikely) for Write to be called repeatedly for an index file.
		// In that case, we'll repeatedly rewrite the oci-layout file, which doesn't hurt,
		// because the content is always identical.
		if err := os.WriteFile(layoutFile, []byte(OciLayoutFileContent), ContentFileMode); err != nil {
			return 0, fmt.Errorf("error writing oci-layout file: %v, Err: %w", layoutFile, err)
		}

		// Now write the index.json file.
		filePath = path.Join(w.Path, OciIndexFileName)
		writeFlags, err := getWriteFlags(filePath)
		if err != nil {
			return 0, fmt.Errorf("error stat'ing file: %v, Err: %w", filePath, err)
		}
		err = writeFile(filePath, writeFlags, b)
		if err != nil {
			return 0, err
		}
	} else {
		// This is a blob. Write it to the blobs folder.
		algo := w.Opts.Desc.Digest.Algorithm()
		blobDir := path.Join(w.Path, BlobsFolderName, algo.String())
		if err := os.MkdirAll(blobDir, ContentFileMode); err != nil {
			return 0, fmt.Errorf("error creating blobDir: %v, Err: %w", blobDir, err)
		}
		filePath = path.Join(blobDir, w.Opts.Desc.Digest.Encoded())

		writeFlags, err := getWriteFlags(filePath)
		if err != nil {
			return 0, fmt.Errorf("error stat'ing file: %v, Err: %w", filePath, err)
		}

		// Now write the blob file.
		err = writeFile(filePath, writeFlags, b)
		if err != nil {
			return 0, err
		}
	}
	fInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("error retrieving FileInfo for file: %v, Err: %w", filePath, err)
	}
	w.Stat.UpdatedAt = fInfo.ModTime()
	return len(b), nil
}

func (w *OciImageLayoutWriter) Close() error {
	return nil
}

// Returns an empty digest until after Commit is called.
func (w *OciImageLayoutWriter) Digest() digest.Digest {
	return w.Dig
}

func (w *OciImageLayoutWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) error {
	w.Dig = w.Opts.Desc.Digest
	return nil
}

func (w *OciImageLayoutWriter) Status() (content.Status, error) {
	return w.Stat, nil
}

func (w *OciImageLayoutWriter) Truncate(size int64) error {
	return errors.New("truncation is unsupported")
}
