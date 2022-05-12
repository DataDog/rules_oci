## `rules_oci` - blazing fast Bazel rules for building OCI Images

RULES_OCI IS HIGHLY EXPERIMENTAL WITH PLANNED BREAKING CHANGES, PLEASE DO NOT
DEPEND ON FOR PRODUCTION USE-CASES.

A Bazel rule-set for extending, creating and publishing OCI artifacts, including image
manifests, image indexes (multi-arch images) and custom artifacts
([ORAS](https://github.com/oras-project)), with a focus on:
* **Speed**, only pulling artifacts that are needed at build-time (no more long image pull times)
* **Extensibility**, creating custom artifacts to leverage standard OCI distribution
  APIs
* **Multi-arch images**, compiling and building multi-arch images with a single Bazel invocation

In addition to Bazel rules, we offer many helpers for interacting with OCI
artifacts under the `pkg` directory and a CLI tool for creating new OCI
artifacts.

`rules_oci` makes an effort to support Docker media types, but there is no
guarantee of long-term support. Most CRI support the OCI types or there are
tools available to convert [between the
specifications](https://github.com/opencontainers/image-spec/blob/v1.0.2/conversion.md).

### Setup

```
# Load OCI Bootstrapping rules or copy the rule into your repository.
git_repository(
    name = "rules_oci_bootstrap",
    remote = "https://github.com/DataDog/rules_oci_bootstrap.git",
    commit = "bab1f71790b74ee78c1d308854ca8e1f23265f94",
)

load("@rules_oci_bootstrap//:defs.bzl", "oci_blob_pull")
oci_blob_pull(
    name = "com_github_datadog_rules_oci",
    digest = "sha256:cc6c59ed7da6bb376552461e06068f883bbe335359c122c15dce3c24e19cd8e2",
    extract = True,
    registry = "ghcr.io",
    repository = "datadog/rules_oci/rules",
    type = "tar.gz",
)
```

### Docs

[Rule API](docs/docs.md)

Examples can be found in the `tests` directory.

### How it works at a high level

At fetch-time we only pull down the manifest json that represents the
structure of the image, rather than pull down everything -- we call this a shallow
pull. We then modify the manifest and republish it with just the changed layers
at "bazel run"-time.

This is perfect for the use-case of creating "application images", aka images
where you just plop a binary on top of a base image. Some additional small
changes can be done such as injecting a shared library or a config file.

We've found in most cases we don't need to pull these additional layers as they
were pushed there previously or can copy (via the mount api) within the same
registry.

This has the downside that there is no verification of all of the content
in the image, but this trade-off is worth the speed of not downloaded many GBs of
base images.

### Roadmap
* [ ] Flesh out code for non-shallow pulls and cases where the layers are coming
      from a different registry.
* [ ] Full Starlark DSL for creating custom artifacts, it's currently looks
  a bit wonky
* [ ] Support for the ORAS Artifact Spec
* [ ] Support for custom artifact crawlers to pull artifacts that have children
not represented by the OCI Image Spec. Ex pulling a full CNAB bundle and all
dependencies.
* [ ] Benchmark against `rules_docker` and raw `docker build`.

### FAQ

**Comparison to `rules_docker`**
* `rules_docker` is built on `go-containerregistry`, which is focused on Docker,
  `rules_oci` uses `containerd` whose implementation complies more to the OCI spec
  and more easily supports custom artifacts
* `rules_oci` focused on supporting the OCI Image spec, rather than the Docker
  spec
* `rules_oci` doesn't have language specific rules, instead a higher-level
  package can build on `rules_oci` to create rules like `go_image`
* `rules_docker` doesn't have support for multi-arch images [#1599](https://github.com/bazelbuild/rules_docker/issues/1599)
