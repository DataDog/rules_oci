package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"

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

	digester := digest.SHA256.Digester()
	wc := ociutil.NewWriterCounter(io.MultiWriter(out, digester.Hash()))
	tw := tar.NewWriter(wc)
	defer tw.Close()

	for _, filePath := range files {
		err = tarutil.AppendFileToTarWriter(filePath, dir, tw)
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

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayer,
		Size:      int64(wc.Count()),
		Digest:    digester.Digest(),
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	return nil
}
