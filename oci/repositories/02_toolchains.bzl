""" toolchains """

load("@aspect_bazel_lib//lib:repositories.bzl", "aspect_bazel_lib_dependencies", "register_coreutils_toolchains")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@rules_rust//crate_universe:defs.bzl", "crates_repository")
load("@rules_rust//rust:repositories.bzl", "rules_rust_dependencies", "rust_register_toolchains")

def rules_oci_toolchains(
        register_go_toolchain_version = None,  # str | None
        register_rust_toolchain_version = None):  # str | None
    """Register toolchains and dependencies for rules_oci

    Args:
        register_go_toolchain_version: The version of the Go toolchain to register.
          Leave this as None if you already depend on rules_go and have already
          registered a go toolchain.
        register_rust_toolchain_version: The version of the Rust toolchain to register.
          Leave this as None if you already depend on rules_rust and have already
          registered a rust toolchain
    """

    # aspect_bazel_lib
    aspect_bazel_lib_dependencies()

    register_coreutils_toolchains(register = True)

    # go
    go_rules_dependencies()

    if register_go_toolchain_version != None:
        go_register_toolchains(version = register_go_toolchain_version)

    # gazelle
    gazelle_dependencies()

    # rust
    rules_rust_dependencies()

    if register_rust_toolchain_version != None:
        rust_register_toolchains()

    crates_repository(
        name = "com_github_datadog_rules_oci_crate_index",
        cargo_lockfile = "@com_github_datadog_rules_oci//:Cargo.lock",
        manifests = [
            "@com_github_datadog_rules_oci//:Cargo.toml",
            "@com_github_datadog_rules_oci//ocitool:Cargo.toml",
        ],
        rust_version = register_rust_toolchain_version,
    )
