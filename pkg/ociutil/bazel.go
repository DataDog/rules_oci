// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bazelbuild/bazel-gazelle/rule"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// GenerateBuildFilesHandler generates build files while walking a tree.
// TODO Ideally, this should actually be a content.WalkFunc, but ocilayout doesn't
// implement this interface yet
func GenerateBuildFilesHandler(handler images.HandlerFunc, layoutRoot string, provider content.Provider) images.HandlerFunc {
	blobBuildFiles := make(map[digest.Algorithm]*rule.File)
	var writemx sync.Mutex

	// TODO Currently only supporting SHA256
	blobBuildFiles[digest.SHA256] = rule.EmptyFile(algoBUILDPath(layoutRoot, digest.SHA256), "")

	// Add load statements for all of the oci_* rules
	ldBlob := rule.NewLoad("@com_github_datadog_rules_oci//oci:blob.bzl")
	ldBlob.Add("oci_blob")

	ldManifest := rule.NewLoad("@com_github_datadog_rules_oci//oci:manifests.bzl")
	ldManifest.Add("oci_image_index_manifest")
	ldManifest.Add("oci_image_manifest")

	for algo, f := range blobBuildFiles {
		ldBlob.Insert(f, 0)
		ldManifest.Insert(f, 0)
		f.Save(algoBUILDPath(layoutRoot, algo))
	}

	// Top level build file for used as an index of the entire layout
	layoutBuild := rule.EmptyFile(filepath.Join(layoutRoot, "BUILD.bazel"), "")

	ldLayout := rule.NewLoad("@com_github_datadog_rules_oci//oci:layout.bzl")
	ldLayout.Add("oci_layout_index")
	ldLayout.Insert(layoutBuild, 0)

	indexRule := rule.NewRule("oci_layout_index", "layout")
	indexRule.SetAttr("visibility", PublicVisibility)
	indexRule.Insert(layoutBuild)

	layoutBuild.Save(filepath.Join(layoutRoot, "BUILD.bazel"))

	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		writemx.Lock()
		defer writemx.Unlock()

		if !blobExists(layoutRoot, desc.Digest) {
			return nil, images.ErrSkipDesc
		}

		algo := desc.Digest.Algorithm()
		f, ok := blobBuildFiles[algo]
		if !ok {
			return nil, fmt.Errorf("no build file for algo '%v'", algo)
		}

		// Insert a rule for each blob
		blobRuleFromDescriptor(desc).Insert(f)

		// if the manifest is an manifest or index, add an additional rule to
		// complete graph
		switch desc.MediaType {
		case ocispec.MediaTypeImageManifest, images.MediaTypeDockerSchema2Manifest:
			manifest, err := ImageManifestFromProvider(ctx, provider, desc)
			if err != nil {
				return nil, err
			}

			imageManifestRule(desc, manifest).Insert(f)
			break
		case ocispec.MediaTypeImageIndex, images.MediaTypeDockerSchema2ManifestList:
			index, err := ImageIndexFromProvider(ctx, provider, desc)
			if err != nil {
				return nil, err
			}

			imageIndexManifestRule(desc, index).Insert(f)
			break
		}

		// Save all BUILD files
		for algo, bf := range blobBuildFiles {
			err := bf.Save(algoBUILDPath(layoutRoot, algo))
			if err != nil {
				return nil, err
			}
		}

		ldLayout.Insert(layoutBuild, 0)
		indexRule.SetAttr("blobs", append(indexRule.AttrStrings("blobs"), dgstToLabel(desc.Digest)))
		err := layoutBuild.Save(filepath.Join(layoutRoot, "BUILD.bazel"))
		if err != nil {
			return nil, err
		}

		return handler(ctx, desc)
	}
}

func blobExists(layoutRoot string, dgst digest.Digest) bool {
	_, err := os.Stat(descToFilePath(layoutRoot, dgst))
	if os.IsNotExist(err) {
		return false
	}

	return true
}

func descToFilePath(root string, dgst digest.Digest) string {
	return filepath.Join(root, "blobs", dgst.Algorithm().String(), dgst.Encoded())
}

func algoBUILDPath(root string, algo digest.Algorithm) string {
	return filepath.Join(root, "blobs", algo.String(), "BUILD.bazel")
}

func dgstToManifestLabelName(dgst digest.Digest) string {
	return fmt.Sprintf("manifest-%v-%v", dgst.Algorithm().String(), dgst.Encoded())
}

func dgstToLabelName(dgst digest.Digest) string {
	return fmt.Sprintf("%v-%v", dgst.Algorithm().String(), dgst.Encoded())
}

func dgstToLabel(dgst digest.Digest) string {
	return fmt.Sprintf("//blobs/%s:%s", dgst.Algorithm().String(), dgstToLabelName(dgst))
}

func descriptorListToLabels(desc []ocispec.Descriptor) []string {
	layerTargets := make([]string, 0, len(desc))
	for _, desc := range desc {
		layerTargets = append(layerTargets, dgstToLabel(desc.Digest))
	}

	return layerTargets
}

var (
	PublicVisibility = []string{"//visibility:public"}
)

func blobRuleFromDescriptor(desc ocispec.Descriptor) *rule.Rule {
	r := rule.NewRule("oci_blob", dgstToLabelName(desc.Digest))
	r.SetAttr("file", desc.Digest.Encoded())
	r.SetAttr("digest", desc.Digest.String())
	r.SetAttr("media_type", desc.MediaType)
	r.SetAttr("size", desc.Size)
	r.SetAttr("annotations", desc.Annotations)
	r.SetAttr("urls", desc.URLs)
	r.SetAttr("visibility", PublicVisibility)

	return r
}

func imageManifestRule(desc ocispec.Descriptor, manifest ocispec.Manifest) *rule.Rule {
	r := rule.NewRule("oci_image_manifest", dgstToManifestLabelName(desc.Digest))

	r.SetAttr("descriptor", dgstToLabel(desc.Digest))
	r.SetAttr("config", dgstToLabel(manifest.Config.Digest))
	// TODO(griffin) Not handling shallow well
	//r.SetAttr("layers", descriptorListToLabels(manifest.Layers))
	r.SetAttr("annotations", manifest.Annotations)
	r.SetAttr("visibility", PublicVisibility)
	r.SetAttr("layout", "//:layout")

	return r
}

func imageIndexManifestRule(desc ocispec.Descriptor, manifest ocispec.Index) *rule.Rule {
	r := rule.NewRule("oci_image_index_manifest", dgstToManifestLabelName(desc.Digest))

	r.SetAttr("descriptor", dgstToLabel(desc.Digest))
	r.SetAttr("manifests", descriptorListToLabels(manifest.Manifests))
	r.SetAttr("annotations", manifest.Annotations)
	r.SetAttr("visibility", PublicVisibility)
	r.SetAttr("layout", "//:layout")

	return r
}

func blobPath(layoutRoot string, dgst digest.Digest) string {
	return filepath.Join(layoutRoot, dgst.Algorithm().String())
}
