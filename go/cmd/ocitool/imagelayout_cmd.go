package main

import (
	"fmt"
	"path"

	"github.com/DataDog/rules_oci/go/pkg/ociutil"
	"github.com/containerd/containerd/images"
	"github.com/urfave/cli/v2"
)

// This command creates an OCI Image Layout directory based on the layout parameter.
// See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
// for the structure of OCI Image Layout directories.
func CreateOciImageLayoutCmd(c *cli.Context) error {
	registry := c.String("registry")
	repository := c.String("repository")
	ref := path.Join(registry, repository)

	// Setup an OCI resolver. We need this because the provided input layout
	// may not contain all required blobs locally. The missing blobs must be
	// loaded from a registry.
	resolver := ociutil.DefaultResolver()
	// Get the fetcher from the resolver, and convert to a provider.
	remoteFetcher, err := resolver.Fetcher(c.Context, ref)
	if err != nil {
		return err
	}
	ociProvider := ociutil.FetchertoProvider(remoteFetcher)

	// Load providers that read local files, and create a multiprovider that
	// contains all of them, as well as the ociProvider.
	providers, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}
	// If modifying the code below, ensure ociProvider comes after providers...
	// we want to use the local provider if a descriptor is present in both.
	providers = append(providers, ociProvider)
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
