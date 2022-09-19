package main

import (

    "github.com/DataDog/rules_oci/pkg/ociutil"
    "github.com/DataDog/rules_oci/internal/flagutil"

    "github.com/urfave/cli/v2"
)

func CreateBlobCmd(c *cli.Context) error {
	file := c.String("file")

    desc, fd, err := ociutil.CreateDescriptorFromFile(file)
    if err != nil {
        return err
    }
    fd.Close()

    desc.MediaType = c.String("media-type")
    desc.Annotations = c.Generic("annotations").(*flagutil.KeyValueFlag).Map

    err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	if err != nil {
		return err
	}

    return nil
}
