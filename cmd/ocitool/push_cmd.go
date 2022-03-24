package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/internal/flagutil"
	"github.com/DataDog/rules_oci/pkg/ociutil"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/urfave/cli/v2"
)

func PushCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

	allProviders := ociutil.MultiProvider(localProviders...)

	baseDesc, err := ociutil.ReadDescriptorFromFile(c.String("desc"))
	if err != nil {
		return fmt.Errorf("failed to read base descriptor: %w", err)
	}

	headers := c.Generic("headers").(*flagutil.KeyValueFlag).Map
	if headers == nil {
		headers = map[string]string{}
	}
	// tack on the X-Meta- prefix
	for k, v := range c.Generic("x_meta_headers").(*flagutil.KeyValueFlag).Map {
		headers["X-Meta-"+k] = v
	}

	resolver := ociutil.ResolverWithHeaders(headers)

	ref := c.String("target-ref")

	pusher, err := resolver.Pusher(c.Context, ref)
	if err != nil {
		return fmt.Errorf("failed to create pusher: %w", err)
	}

	regIng, ok := pusher.(content.Ingester)
	if !ok {
		return fmt.Errorf("pusher not an ingester: %T", pusher)
	}

	switch baseDesc.MediaType {
	// We only want do a shallow push for indexes.
	// This allows us to use separate tags for the index vs the child images
	case images.MediaTypeDockerSchema2ManifestList, ocispec.MediaTypeImageIndex:
		err = ociutil.CopyContent(c.Context, allProviders, regIng, baseDesc)
	default:
		imagesHandler := images.ChildrenHandler(allProviders)
		err = ociutil.CopyContentFromHandler(c.Context, imagesHandler, allProviders, regIng, baseDesc)
	}

	if err != nil {
		return fmt.Errorf("failed to push content to registry: %w", err)
	}

	fmt.Printf("Reference: %v@%v\n", ref, baseDesc.Digest.String())

	return nil
}
