package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/DataDog/rules_oci/go/pkg/blob"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
	"github.com/urfave/cli/v2"
)

// Given a slice of layoutFilePaths, where each path contains a file that may
// be used within an OCI Image Format, return an index that maps sha256 values
// to paths.
// If relPath is non-empty, it is prepended to all layoutFilePaths.
func getLayoutFilesBlobIndex(layoutFilePaths []string, relPath string) (blob.Index, error) {
	var result blob.Index
	result.Blobs = make(map[digest.Digest]string)
	for _, p := range layoutFilePaths {
		if len(strings.TrimSpace(p)) == 0 {
			// Ignore empty paths.
			continue
		}
		if relPath != "" {
			p = path.Join(relPath, p)
		}
		// Use an immediately invoked function here so that defer closes the
		// file at a suitable time.
		err := func() error {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()
			digester := digest.SHA256.Digester()
			_, err = f.WriteTo(digester.Hash())
			if err != nil {
				return err
			}
			digest := digester.Digest()
			result.Blobs[digest] = p
			return nil
		}()
		if err != nil {
			return blob.Index{}, err
		}

	}

	return result, nil
}

// This command creates an OCI Image Layout directory based on the layout parameter.
// See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
// for the structure of OCI Image Layout directories.
func CreateOciImageLayoutCmd(c *cli.Context) error {
	relPath := c.String("layout-relative")
	// Load providers that read local files, and create a multiprovider that
	// contains all of them, as well as providers for base image blobs.
	providers, err := LoadLocalProviders(c.StringSlice("layout"), relPath)
	if err != nil {
		return err
	}

	layoutFilesBlobIdx, err := getLayoutFilesBlobIndex(c.StringSlice("layout-files"), relPath)
	if err != nil {
		return err
	}

	if len(layoutFilesBlobIdx.Blobs) > 0 {
		providers = append(providers, &layoutFilesBlobIdx)
	}

	multiProvider := ociutil.MultiProvider(providers...)

	descriptorFile := c.String("desc")
	baseDesc, err := ociutil.ReadDescriptorFromFile(descriptorFile)
	if err != nil {
		return fmt.Errorf("failed to read base descriptor: %w", err)
	}

	outDir := c.String("out-dir")
	ociIngester, err := ociutil.NewOciImageLayoutIngester(outDir)
	if err != nil {
		return err
	}

	// Copy the children first; leave the parent (index) to last.
	imagesHandler := images.ChildrenHandler(multiProvider)
	err = ociutil.CopyChildrenFromHandler(c.Context, imagesHandler, multiProvider, &ociIngester, baseDesc)
	if err != nil {
		return fmt.Errorf("failed to copy child content to OCI Image Layout: %w", err)
	}

	// copy the parent last (in case of image index)
	err = ociutil.CopyContent(c.Context, multiProvider, &ociIngester, baseDesc)
	if err != nil {
		return fmt.Errorf("failed to copy parent content to OCI Image Layout: %w", err)
	}

	return nil
}
