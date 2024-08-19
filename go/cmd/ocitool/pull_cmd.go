package main

import (
	"golang.org/x/sync/semaphore"

	"github.com/DataDog/rules_oci/go/pkg/ociutil"

	"github.com/containerd/containerd/images"
	"github.com/containerd/log"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/urfave/cli/v2"
	orascontent "oras.land/oras-go/pkg/content"
)

func PullCmd(c *cli.Context) error {
	ref := c.Args().First()

	ctx := log.WithLogger(c.Context, log.G(c.Context).WithField("pull-ref", ref))

	resolver := ociutil.DefaultResolver()

	name, desc, err := resolver.Resolve(ctx, ref)
	if err != nil {
		return err
	}

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}

	desc.Annotations[ocispec.AnnotationRefName] = name

	log.G(ctx).
		WithField("name", name).
		WithField("desc", desc).
		Debug("resolved")

	layoutPath := c.StringSlice("layout")[0]

	layout, err := orascontent.NewOCI(layoutPath)
	if err != nil {
		return err
	}

	remoteFetcher, err := resolver.Fetcher(ctx, name)
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

	err = images.Dispatch(ctx, ociutil.CopyContentHandler(imagesHandler, provider, layout), sem, desc)
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
