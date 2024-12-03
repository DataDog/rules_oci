package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/internal/tarutil"
	"github.com/DataDog/rules_oci/go/pkg/layer"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/DataDog/zstd"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
)

func CreateLayerCmd(c *cli.Context) error {
	config, err := parseConfig(c)
	if err != nil {
		return fmt.Errorf("problem parsing config: %w", err)
	}

	dir := config.Directory
	files := config.Files

	out, err := os.Create(config.OutputLayer)
	if err != nil {
		return err
	}

	digester := digest.SHA256.Digester()
	wc := ociutil.NewWriterCounter(io.MultiWriter(out, digester.Hash()))
	var tw *tar.Writer
	var zstdWriter *zstd.Writer
	var gzipWriter *gzip.Writer
	var mediaType string

	if config.UseZstd {
		zstdWriter = zstd.NewWriter(wc)
		mediaType = ocispec.MediaTypeImageLayerZstd
		tw = tar.NewWriter(zstdWriter)
	} else {
		gzipWriter = gzip.NewWriter(wc)
		gzipWriter.Name = path.Base(out.Name())
		mediaType = ocispec.MediaTypeImageLayerGzip
		tw = tar.NewWriter(gzipWriter)
	}

	defer tw.Close()

	for _, filePath := range files {
		err = tarutil.AppendFileToTarWriter(filePath, filepath.Join(dir, filepath.Base(filePath)), tw)
		if err != nil {
			return err
		}
	}

	for filePath, storePath := range config.FileMapping {
		err = tarutil.AppendFileToTarWriter(filePath, storePath, tw)
		if err != nil {
			return err
		}
	}

	for k, v := range config.SymlinkMapping {
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
	if config.UseZstd {
		zstdWriter.Close()
	} else {
		gzipWriter.Close()
	}

	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      int64(wc.Count()),
		Digest:    digester.Digest(),
	}

	bazelLabel := config.BazelLabel
	if bazelLabel != "" {
		desc.Annotations = map[string]string{
			// This will also be added to the image config layer history by append-layers
			layer.AnnotationArtifactDescription: bazelLabel,
		}
	}

	err = ociutil.WriteDescriptorToFile(config.Descriptor, desc)
	if err != nil {
		return err
	}

	return nil
}

type createLayerConfig struct {
	BazelLabel     string            `json:"bazel-label" toml:"bazel-label" yaml:"bazel-label"`
	Descriptor     string            `json:"outd" toml:"outd" yaml:"outd"`
	Directory      string            `json:"dir" toml:"dir" yaml:"dir"`
	Files          []string          `json:"file" toml:"file" yaml:"file"`
	FileMapping    map[string]string `json:"file-map" toml:"file-map" yaml:"file-map"`
	OutputLayer    string            `json:"out" toml:"out" yaml:"out"`
	SymlinkMapping map[string]string `json:"symlink" toml:"symlink" yaml:"symlink"`
	UseZstd        bool              `json:"zstd-compression" toml:"zstd-compression" yaml:"zstd-compression"`
}

func newCreateLayerConfig(c *cli.Context) *createLayerConfig {
	return &createLayerConfig{
		BazelLabel:     c.String("bazel-label"),
		Directory:      c.String("dir"),
		Files:          c.StringSlice("file"),
		FileMapping:    c.Generic("file-map").(*flagutil.KeyValueFlag).Map,
		OutputLayer:    c.String("out"),
		Descriptor:     c.String("outd"),
		SymlinkMapping: c.Generic("symlink").(*flagutil.KeyValueFlag).Map,
		UseZstd:        c.Bool("zstd-compression"),
	}
}

func parseConfig(c *cli.Context) (*createLayerConfig, error) {
	configFile := c.Path("configuration-file")
	if configFile == "" {
		return newCreateLayerConfig(c), nil
	}

	file, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("problem reading config file: %w", err)
	}

	var config createLayerConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("problem parsing config file as JSON: %w", err)
	}

	return &config, nil
}
