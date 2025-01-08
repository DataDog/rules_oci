package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/go/pkg/ociutil"

	"github.com/urfave/cli/v2"
)

func PushBlobCmd(c *cli.Context) error {
	resolver := ociutil.DefaultResolver()

	desc, err := resolver.PushBlob(c.Context, c.String("file"), c.String("ref"), "")
	if err != nil {
		return fmt.Errorf("failed to push blob: %w", err)
	}

	url, err := ociutil.DescriptorToURL(c.String("ref"), desc)
	if err != nil {
		return err
	}

	fmt.Printf("Pushed: %v", url)

	return nil
}
