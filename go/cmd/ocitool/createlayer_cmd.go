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
	"strconv"
	"strings"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/internal/tarutil"
	"github.com/DataDog/rules_oci/go/pkg/layer"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
)

func CreateLayerCmd(c *cli.Context) error {
	config, err := parseConfig(c)
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	out, err := os.Create(config.OutputLayer)
	if err != nil {
		return fmt.Errorf("failed to create file at %s: %w", config.OutputLayer, err)
	}

	digester := digest.SHA256.Digester()
	wc := ociutil.NewWriterCounter(io.MultiWriter(out, digester.Hash()))
	gw := gzip.NewWriter(wc)
	gw.Name = path.Base(out.Name())
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, hostPath := range config.Files {
		tarPath := filepath.Join(
			config.Directory,
			filepath.Base(hostPath),
		)

		tarUid, err := config.uid(tarPath)
		if err != nil {
			return err
		}

		tarGid, err := config.gid(tarPath)
		if err != nil {
			return err
		}

		err = tarutil.AppendFileToTarWriter(
			/* hostPath */ hostPath,
			/* tarPath  */ tarPath,
			/* tarMode  */ config.mode(tarPath),
			/* tarUid   */ tarUid,
			/* tarGid   */ tarGid,
			/* tw       */ tw,
		)

		if err != nil {
			return err
		}
	}

	for hostPath, tarPath := range config.FileMapping {
		tarUid, err := config.uid(tarPath)
		if err != nil {
			return err
		}

		tarGid, err := config.gid(tarPath)
		if err != nil {
			return err
		}

		err = tarutil.AppendFileToTarWriter(
			/* hostPath */ hostPath,
			/* tarPath  */ tarPath,
			/* tarMode  */ config.mode(tarPath),
			/* tarUid   */ tarUid,
			/* tarGid   */ tarGid,
			/* tw       */ tw,
		)
		if err != nil {
			return err
		}
	}

	for tarPath, tarTarget := range config.SymlinkMapping {
		tarUid, err := config.uid(tarPath)
		if err != nil {
			return err
		}

		tarGid, err := config.gid(tarPath)
		if err != nil {
			return err
		}

		err = tarutil.AppendSymlinkToTarWriter(
			/* tarPath   */ tarPath,
			/* tarTarget */ tarTarget,
			/* tarMode   */ config.mode(tarPath),
			/* tarUid    */ tarUid,
			/* tarGid    */ tarGid,
			/* tw        */ tw,
		)
		if err != nil {
			return err
		}

	}

	// Need to flush before we count bytes and digest, might as well close since
	// it's not needed anymore.
	tw.Close()
	gw.Close()

	desc := ocispec.Descriptor{
		Digest:    digester.Digest(),
		MediaType: ocispec.MediaTypeImageLayerGzip,
		Size:      int64(wc.Count()),
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
	FileMapping    map[string]string `json:"file-map" toml:"file-map" yaml:"file-map"`
	Files          []string          `json:"file" toml:"file" yaml:"file"`
	ModeMapping    map[string]int64  `json:"mode-map" toml:"mode-map" yaml:"mode-map"`
	OutputLayer    string            `json:"out" toml:"out" yaml:"out"`
	OwnerMapping   map[string]string `json:"owner-map" toml:"owner-map" yaml:"owner-map"`
	SymlinkMapping map[string]string `json:"symlink" toml:"symlink" yaml:"symlink"`
}

func newCreateLayerConfig(c *cli.Context) (*createLayerConfig, error) {
	modeMapping := make(map[string]int64)
	for path, modeStr := range c.Generic("mode-map").(*flagutil.KeyValueFlag).Map {
		mode, err := strconv.ParseInt(modeStr, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing mode-map value %s: %w", modeStr, err)
		}
		modeMapping[path] = mode
	}
	return &createLayerConfig{
		BazelLabel:     c.String("bazel-label"),
		Descriptor:     c.String("outd"),
		Directory:      c.String("dir"),
		FileMapping:    c.Generic("file-map").(*flagutil.KeyValueFlag).Map,
		Files:          c.StringSlice("file"),
		ModeMapping:    modeMapping,
		OutputLayer:    c.String("out"),
		OwnerMapping:   c.Generic("owner-map").(*flagutil.KeyValueFlag).Map,
		SymlinkMapping: c.Generic("symlink").(*flagutil.KeyValueFlag).Map,
	}, nil
}

func parseConfig(c *cli.Context) (*createLayerConfig, error) {
	configFile := c.Path("configuration-file")
	if configFile == "" {
		config, err := newCreateLayerConfig(c)
		if err != nil {
			return nil, err
		}
		return config, nil
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

func (c *createLayerConfig) mode(tarPath string) int64 {
	if i, exists := c.ModeMapping[tarPath]; exists {
		return i
	}
	return 0
}

func (c *createLayerConfig) uid(tarPath string) (*int, error) {
	if s, exists := c.OwnerMapping[tarPath]; exists {
		s := strings.SplitN(s, ":", 2)[0]
		n, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing uid %s: %w", s, err)
		}
		i := int(n)
		return &i, nil
	}
	return nil, nil
}

func (c *createLayerConfig) gid(tarPath string) (*int, error) {
	if s, exists := c.OwnerMapping[tarPath]; exists {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) > 1 {
			s := parts[1]
			n, err := strconv.ParseInt(s, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("error parsing gid %s: %w", s, err)
			}
			i := int(n)
			return &i, nil
		}
	}
	return nil, nil
}
