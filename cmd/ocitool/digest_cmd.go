package main

import (
	"os"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/urfave/cli/v2"
)

func DigestCmd(c *cli.Context) error {
	desc, err := ociutil.ReadDescriptorFromFile(c.String("desc"))
	if err != nil {
		return err
	}

	err = os.WriteFile(c.String("out"), []byte(desc.Digest.String()), 0655)
	if err != nil {
		return err
	}

	return nil
}
