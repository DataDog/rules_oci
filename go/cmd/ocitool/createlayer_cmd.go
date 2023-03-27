package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/internal/tarutil"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
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
	gw := gzip.NewWriter(wc)
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

	desc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageLayerGzip,
		Size:      int64(wc.Count()),
		Digest:    digester.Digest(),
	}

	bazelLabel := c.String("bazel-label")
	if bazelLabel != "" {
		desc.Annotations = map[string]string{
			// This will also be added to the image config layer history by append-layers
			ocispec.AnnotationArtifactDescription: bazelLabel,
		}
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

	return nil
}
