package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/pkg/ociutil"

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
		return err
	}

	resolver := ociutil.NewDDRegistryResolver()

	ref := c.String("target-ref")

	pusher, err := resolver.Pusher(c.Context, ref)
	if err != nil {
		return err
	}

	regIng, ok := pusher.(content.Ingester)
	if !ok {
		return fmt.Errorf("pusher not an ingester: %T", pusher)
	}

	imagesHandler := images.ChildrenHandler(allProviders)

	err = ociutil.CopyContentFromHandler(c.Context, imagesHandler, allProviders, regIng, baseDesc)
	if err != nil {
		return err
	}

	fmt.Printf("Reference: %v@%v\n", ref, baseDesc.Digest.String())

	return nil
}
