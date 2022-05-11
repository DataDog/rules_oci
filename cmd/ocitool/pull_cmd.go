// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package main

import (
	"golang.org/x/sync/semaphore"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	orascontent "oras.land/oras-go/pkg/content"
)

func PullCmd(c *cli.Context) error {
	resolver := ociutil.DefaultResolver()

	ref := c.Args().First()

	name, desc, err := resolver.Resolve(c.Context, ref)
	if err != nil {
		return err
	}

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}

	desc.Annotations[ocispec.AnnotationRefName] = name

	log.Debugf("Resolved descriptor for %v: %#v", name, desc)

	layoutPath := c.StringSlice("layout")[0]

	layout, err := orascontent.NewOCI(layoutPath)
	if err != nil {
		return err
	}

	log.Debugf("found layout at '%v'", layoutPath)

	remoteFetcher, err := resolver.Fetcher(c.Context, name)
	if err != nil {
		return err
	}

	provider := ociutil.FetchertoProvider(remoteFetcher)

	sem := semaphore.NewWeighted(int64(c.Uint("parallel")))

	imagesHandler := images.ChildrenHandler(provider)
	if c.Bool("shallow") {
		imagesHandler = ociutil.ContentTypesFilterHandler(imagesHandler,
			ocispec.MediaTypeImageManifest,
			ocispec.MediaTypeImageIndex,
			ocispec.MediaTypeImageConfig,
			images.MediaTypeDockerSchema2Manifest,
			images.MediaTypeDockerSchema2ManifestList,
			images.MediaTypeDockerSchema2Config,
		)
	}

	err = images.Dispatch(c.Context, ociutil.CopyContentHandler(imagesHandler, provider, layout), sem, desc)
	if err != nil {
		return err
	}

	layout.AddReference(name, desc)
	err = layout.SaveIndex()
	if err != nil {
		return err
	}

	return nil
}
