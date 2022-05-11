// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DataDog/rules_oci/pkg/ociutil"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	orascontent "oras.land/oras-go/pkg/content"
)

func GenerateBuildFilesCmd(c *cli.Context) error {
	allLocalLayoutsPaths := c.StringSlice("layout")
	if len(allLocalLayoutsPaths) > 1 {
		return fmt.Errorf("too many layouts")
	} else if len(allLocalLayoutsPaths) <= 0 {
		return fmt.Errorf("need at least one layout")
	}

	layoutRootPath := allLocalLayoutsPaths[0]

	layout, err := orascontent.NewOCI(layoutRootPath)
	if err != nil {
		return err
	}

	refs := layout.ListReferences()
	refDescs := make([]ocispec.Descriptor, 0, len(refs))

	for _, r := range refs {
		refDescs = append(refDescs, r)
	}

	log.Debugf("layout root: %#v", refs)

	err = images.Walk(
		context.Background(),
		ociutil.GenerateBuildFilesHandler(images.ChildrenHandler(layout), layoutRootPath, layout),
		refDescs...,
	)
	if err != nil {
		return err
	}

	imageTargetDigest := c.String("image-digest")
	if imageTargetDigest != "" {
		err = os.MkdirAll(filepath.Join(layoutRootPath, "image"), 0700)
		if err != nil {
			return err
		}

		imageTargetBuildFilePath := filepath.Join(layoutRootPath, "image", "BUILD.bazel")
		imageTargetBuild := rule.EmptyFile(imageTargetBuildFilePath, "")

		aliasRule := rule.NewRule("alias", "image")
		aliasRule.SetAttr("actual", dgstToManifestLabel(digest.Digest(imageTargetDigest)))
		aliasRule.SetAttr("visibility", ociutil.PublicVisibility)
		aliasRule.Insert(imageTargetBuild)

		err = imageTargetBuild.Save(imageTargetBuildFilePath)
		if err != nil {
			return err
		}

		log.Debugf("Created BUILD file in image package")
	}

	log.Debugf("Done generating build files")

	return nil
}

// TODO redeclared a couple other places
func dgstToManifestLabel(dgst digest.Digest) string {
	return fmt.Sprintf("//blobs/%s:%s", dgst.Algorithm().String(), dgstToManifestLabelName(dgst))
}

func dgstToManifestLabelName(dgst digest.Digest) string {
	return fmt.Sprintf("%v-%v-%v", "manifest", dgst.Algorithm().String(), dgst.Encoded())
}
