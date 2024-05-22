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

// Given a slice of baseLayoutPaths, where each path contains an OCI Image Format,
// return an index that maps sha256 values to paths.
// If relPath is non-empty, it is prepended to all baseLayoutPaths.
func getBaseLayoutBlobIndex(baseLayoutPaths []string, relPath string) (blob.Index, error) {
	var result blob.Index
	result.Blobs = make(map[digest.Digest]string)

	for _, p := range baseLayoutPaths {
		if len(strings.TrimSpace(p)) == 0 {
			// Ignore empty paths.
			continue
		}
		blobsDir := path.Join(p, "blobs", "sha256")
		if relPath != "" {
			blobsDir = path.Join(relPath, blobsDir)
		}
		entries, err := os.ReadDir(blobsDir)
		if err != nil {
			return blob.Index{}, fmt.Errorf("unable to read OCI Image Format blobs dir. Base layout paths: %v, Relpath: %s, Path: %s, Error: %w", baseLayoutPaths, relPath, blobsDir, err)
		}
		for _, entry := range entries {
			if !entry.Type().IsRegular() {
				continue
			}
			name := entry.Name()
			result.Blobs[digest.Digest(name)] = path.Join(blobsDir, name)
		}
	}

	return result, nil
}

// This command creates an OCI Image Layout directory based on the layout parameter.
// See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
// for the structure of OCI Image Layout directories.
func CreateOciImageLayoutCmd(c *cli.Context) error {
	relPath := c.String("layout-relative")
	baseLayoutBlobIdx, err := getBaseLayoutBlobIndex(c.StringSlice("base-image-layouts"), relPath)
	if err != nil {
		return err
	}

	// Load providers that read local files, and create a multiprovider that
	// contains all of them, as well as providers for base image blobs.
	providers, err := LoadLocalProviders(c.StringSlice("layout"), relPath)
	if err != nil {
		return err
	}

	if len(baseLayoutBlobIdx.Blobs) > 0 {
		providers = append(providers, &baseLayoutBlobIdx)
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
