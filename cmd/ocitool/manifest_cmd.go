package main

import (
	"github.com/urfave/cli/v2"
)

func CreateImageManifestCmd(c *cli.Context) error {
	/*
	       localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	   	if err != nil {
	   		return err
	   	}

	   	bi, err := blob.MergeIndex(localProviders...)
	   	if err != nil {
	   		return err
	   	}



	   	for _, descPath := range descriptorPaths {
	   		desc, err := ociutil.ReadDescriptorFromFile(descPath)
	   		if err != nil {
	   			return fmt.Errorf("failed to load descriptors for index: %w", err)
	   		}

	           descriptors = append(descriptors, desc)
	   	}

	   	err = ociutil.WriteDescriptorToFile(c.String("outd"), desc)
	   	if err != nil {
	   		return err
	   	}

	   	// Append image index to blob index
	   	bi.Blobs[desc.Digest] = c.String("out-index")

	   	err = bi.WriteToFile(c.String("out-layout"))
	   	if err != nil {
	   		return err
	   	}
	*/
	return nil
}
