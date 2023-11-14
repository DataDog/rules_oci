package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"

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

	plainHTTPHost := c.String("plain-http-host")

	resolver := ociutil.ResolverWithHeaders(headers, plainHTTPHost)

	ref := c.String("target-ref")

	pusher, err := resolver.Pusher(c.Context, ref)
	if err != nil {
		return fmt.Errorf("failed to create pusher: %w", err)
	}

	regIng, ok := pusher.(content.Ingester)
	if !ok {
		return fmt.Errorf("pusher not an ingester: %T", pusher)
	}

	// take care of copying any children first
	imagesHandler := images.ChildrenHandler(allProviders)
	err = ociutil.CopyChildrenFromHandler(c.Context, imagesHandler, allProviders, regIng, baseDesc)
	if err != nil {
		return fmt.Errorf("failed to push child content to registry: %w", err)
	}

	// if a tag exists, use it for the parent
	tag := c.String("parent-tag")
	if tag != "" {
		ref = ref + ":" + tag
		pusher, err = resolver.Pusher(c.Context, ref)
		if err != nil {
			return fmt.Errorf("failed to create parent pusher: %w", err)
		}

		regIng, ok = pusher.(content.Ingester)
		if !ok {
			return fmt.Errorf("parent pusher not an ingester: %T", pusher)
		}
	}

	// push the parent last (in case of image index)
	err = ociutil.CopyContent(c.Context, allProviders, regIng, baseDesc)
	if err != nil {
		return fmt.Errorf("failed to push parent content to registry: %w", err)
	}

	fmt.Printf("Reference: %v@%v\n", ref, baseDesc.Digest.String())

	return nil
}
