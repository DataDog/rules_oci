## `rules_oci` - Bazel rules for building OCI Images

RULES_OCI IS HIGHLY EXPERIMENTAL WITH PLANNED BREAKING CHANGES, PLEASE DO NOT
DEPEND ON FOR PRODUCTION USE-CASES.

A Bazel rule-set for extending, creating and publishing OCI artifacts, including image
manifests and image indexes (multi-arch images), with a focus on:

- **Extensibility**, creating custom artifacts to leverage standard OCI distribution APIs
- **Multi-arch images**, compiling and building multi-arch images with a single Bazel invocation

### Setup

```starlark
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "com_github_datadog_rules_oci",
    sha256 = "<SHA256>",
    strip_prefix = "rules_oci-<COMMIT>",
    url = "https://github.com/DataDog/rules_oci/archive/<COMMIT>.tar.gz",
)

load(
    "@com_github_datadog_rules_oci//oci/repositories:01_direct_dependencies.bzl",
    "rules_oci_direct_dependencies",
)

rules_oci_direct_dependencies()

load("@com_github_datadog_rules_oci//oci/repositories:02_toolchains.bzl", "rules_oci_toolchains")

rules_oci_toolchains(
    # Only set this if you have not already registered a go toolchain
    register_go_toolchain_version = "1.22.5",

    # Only set this if you have not already registered a rust toolchain
    register_rust_toolchain_version = "1.82.0",
)

load(
    "@com_github_datadog_rules_oci//oci/repositories:03_third_party_go_and_rust_libraries.bzl",
    "rules_oci_third_party_go_and_rust_libraries",
)

rules_oci_third_party_go_and_rust_libraries()
```

### Docs

- [rules](docs/defs.md)
- [providers](docs/providers.md)
- [pull repository rule](docs/pull.md)

Examples can be found in the `examples` directory.

### FAQ

**Comparison to `rules_docker`**

- `rules_docker` is built on `go-containerregistry`, which is focused on Docker,
  `rules_oci` uses `containerd` whose implementation complies more to the OCI spec
  and more easily supports custom artifacts
- `rules_oci` focused on supporting the OCI Image spec, rather than the Docker
  spec
- `rules_oci` doesn't have language specific rules, instead a higher-level
  package can build on `rules_oci` to create rules like `go_image`
- `rules_docker` doesn't have support for multi-arch images [#1599](https://github.com/bazelbuild/rules_docker/issues/1599)

### Developing

| action                   | command                                                  |
| ------------------------ | -------------------------------------------------------- |
| Run the tests            | `just test`                                              |
| Run the formatter        | `just format`                                            |
| Run gazelle              | `just gazelle`                                           |
| Update the docs          | `just update-docs`                                       |
| Update go dependencies   | Modify `go.mod` and run `just update-go-3rd-party`       |
| Update rust dependencies | Modify `Cargo.toml` and run `just update-rust-3rd-party` |
| Publish a new release    | `just release`                                           |
