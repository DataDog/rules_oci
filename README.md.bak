## `rules_oci` - blazing fast Bazel rules for building OCI Images

A Bazel ruleset for extending, creating and publishing OCI artifacts, including image
manifests, image indexes (multiarch images) and custom artifacts
([ORAS](https://github.com/oras-project)), with a focus on:
* **Speed**, only pulling artifacts that are needed at build-time (no more long image pull times)
* **Extensibility**, creating custom artifacts to leverage standard OCI distribution
  APIs
* **Multiarch images**, compiling and building multiarch images with a single Bazel invocation

In addition to Bazel rules, we offer many helpers for interacting with OCI
artifacts under the `pkg` directory and a CLI tool for creating new OCI
artifacts.

`rules_oci` makes an effort to support Docker media types, but there is no
gaurentee of long-term support. Most CRI support the OCI types or there are
tools available to convert [between the
specifications](https://github.com/opencontainers/image-spec/blob/v1.0.2/conversion.md).

### Docs

[Rule API](docs/docs.md)

Examples can be found in the `tests` directory.

### [WIP] How it works

The gist is that we only pull down the manifest json that represents the
structure of the image, rather than pull down everything -- we call this a shallow
pull. We then modify the manifest and republish it with just the changed layers.

This is perfect for the use-case of creating "application images", aka images
where you just plop a binary on top of a base image. Some additional small
changes can be done such as injecting a shared library or a config file.

This has the downside that there is no verification of all of the content
in the image, but this tradeoff is worth the speed of not downloaded many GBs of
base images.

### Roadmap
* [ ] Resolve limitations
* [ ] Full Starlark DSL for creating custom artifacts, it's currently looks
  a bit wonky
* [ ] Support for the ORAS Artifact Spec
* [ ] Support for custom artifact crawlers to pull artifacts that have children
not represented by the OCI Image Spec. Ex pulling a full CNAB bundle and all
dependencies.

### Current Limitations
* [ ] Non-shallow pulls are sometimes broken
* [ ] Doesn't support docker credential helper authentication
* [ ] Images pulled and pushed must be in the same registry

### FAQ

**Comparison to `rules_docker`**
* `rules_docker` is built on `go-containerregistry`, which is foucssed on Docker,
  `rules_oci` uses `containerd` whose implementation complies more to the OCI spec
  and more easily supports custom artifacts
* `rules_oci` focusses on supporting the OCI Image spec, rather than the Docker
  spec
* `rules_oci` doesn't have language specific rules, instead a higher-level
  package can build on `rules_oci` to create rules like `go_image`
* `rules_docker` doesn't have support for multiarch images [#1599](https://github.com/bazelbuild/rules_docker/issues/1599)
