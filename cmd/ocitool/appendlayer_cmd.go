package main

import (
	"fmt"

	"github.com/DataDog/rules_oci/pkg/blob"
	"github.com/DataDog/rules_oci/pkg/layer"
	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/platforms"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func AppendLayersCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

	allLocalProviders := ociutil.MultiProvider(localProviders...)

	baseDesc, err := ociutil.ReadDescriptorFromFile(c.String("base"))
	if err != nil {
		return err
	}

	targetPlatform := ocispec.Platform{
		OS:           c.String("os"),
		Architecture: c.String("arch"),
	}
	targetPlatformMatch := platforms.Only(targetPlatform)

	var manifestDesc ocispec.Descriptor
	if images.IsIndexType(baseDesc.MediaType) {
		index, err := ociutil.ImageIndexFromProvider(c.Context, allLocalProviders, baseDesc)
		if err != nil {
			return err
		}

		manifestDesc, err = ociutil.ManifestFromIndex(index, targetPlatformMatch)
		if err != nil {
			return err
		}
	} else if images.IsManifestType(baseDesc.MediaType) {
		manifestDesc = baseDesc

		if ociutil.IsEmptyPlatform(manifestDesc.Platform) {
			platform, err := ociutil.ResolvePlatformFromDescriptor(c.Context, allLocalProviders, manifestDesc)
			if err != nil {
				return fmt.Errorf("no platform for base: %w", err)
			}

			manifestDesc.Platform = &platform
		}

	} else {
		return fmt.Errorf("Unknown base image type %q", baseDesc.MediaType)
	}

	if !targetPlatformMatch.Match(*manifestDesc.Platform) {
		return fmt.Errorf("invalid platform, expected %v, recieved %v", targetPlatform, *manifestDesc.Platform)
	}

	if manifestDesc.Annotations == nil {
		manifestDesc.Annotations = make(map[string]string)
	}
	annotations, err := parseAnnotationsFlag(c.String("annotations"))
	if err != nil {
		return err
	}
	for k, v := range annotations {
		manifestDesc.Annotations[k] = v
	}
	baseRef, ok := baseDesc.Annotations[ocispec.AnnotationRefName]
	if ok {
		manifestDesc.Annotations[ocispec.AnnotationRefName] = baseRef
	}

	log.WithField("base_desc", manifestDesc).Debugf("using as base")

	layerPaths := c.StringSlice("layer")

	layerProvider := &blob.Index{
		Blobs: make(map[digest.Digest]string),
	}

	layerDescs := make([]ocispec.Descriptor, 0, len(layerPaths))
	for _, lp := range layerPaths {
		ld, reader, err := ociutil.CreateDescriptorFromFile(lp)
		if err != nil {
			return err
		}
		reader.Close()
		ld.MediaType = images.MediaTypeDockerSchema2Layer

		layerProvider.Blobs[ld.Digest] = lp
		layerDescs = append(layerDescs, ld)
	}

	log.Debugf("created descriptors for layers(n=%v): %#v", len(layerPaths), layerDescs)

	outIngestor := layer.NewAppendIngester(c.String("out-manifest"), c.String("out-config"))

	newManifest, newConfig, err := layer.AppendLayers(
		c.Context,
		ociutil.SplitStore(outIngestor, ociutil.MultiProvider(allLocalProviders, layerProvider)),
		manifestDesc,
		layerDescs,
	)
	if err != nil {
		return err
	}

	layerProvider.Blobs[newManifest.Digest] = c.String("out-manifest")
	layerProvider.Blobs[newConfig.Digest] = c.String("out-config")

	err = layerProvider.WriteToFile(c.String("out-layout"))
	if err != nil {
		return err
	}

	err = ociutil.WriteDescriptorToFile(c.String("outd"), newManifest)
	if err != nil {
		return err
	}

	return nil
}
