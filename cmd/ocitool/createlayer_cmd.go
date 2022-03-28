package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/DataDog/rules_oci/internal/flagutil"
	"github.com/DataDog/rules_oci/internal/tarutil"
	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
)

func CreateLayerCmd(c *cli.Context) error {
	dir := c.String("dir")
	files := c.StringSlice("file")

	out, err := os.Create(c.String("out"))
	if err != nil {
		return err
	}

	gw := gzip.NewWriter(out)
	gw.Name = path.Base(out.Name())
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, filePath := range files {
		err = tarutil.AppendFileToTarWriter(filePath, filepath.Join(dir, filepath.Base(filePath)), tw)
		if err != nil {
			return err
		}
	}

	for filePath, storePath := range c.Generic("file-map").(*flagutil.KeyValueFlag).Map {
		err = tarutil.AppendFileToTarWriter(filePath, storePath, tw)
		if err != nil {
			return err
		}
	}

	for k, v := range c.Generic("symlink").(*flagutil.KeyValueFlag).Map {
		err = tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     k,
			Linkname: v,
		})
		if err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	// Need to flush before we count bytes and digest, might as well close since
	// it's not needed anymore.
	tw.Close()
	gw.Close()

	// grab the digest and size from the files now that they are compressed
	_, err = out.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("unable to reset uncompressed file: %w", err)
	}
	gzDigest, err := digest.SHA256.FromReader(out)
	if err != nil {
		return fmt.Errorf("unable to create diff ID of file: %w", err)
	}
	stats, err := out.Stat()
	if err != nil {
		return err
	}

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayerGzip,
		Size:      stats.Size(),
		Digest:    gzDigest,
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	return nil
}
