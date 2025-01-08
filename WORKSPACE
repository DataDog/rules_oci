workspace(name = "com_github_datadog_rules_oci")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# If this is updated, make sure you also update rust-toolchain.toml
_RUSTC_VERSION = "1.81.0"

http_archive(
    name = "rules_rust",
    integrity = "sha256-r09Wyq5QqZpov845sUG1Cd1oVIyCBLmKt6HK/JTVuwI=",
    urls = ["https://github.com/bazelbuild/rules_rust/releases/download/0.54.1/rules_rust-v0.54.1.tar.gz"],
)

load("@rules_rust//rust:repositories.bzl", "rules_rust_dependencies", "rust_register_toolchains")

rules_rust_dependencies()

rust_register_toolchains(
    edition = "2021",
    extra_target_triples = [
        "aarch64-unknown-linux-gnu",
        "aarch64-apple-darwin",
        "x86_64-apple-darwin",
        "x86_64-unknown-linux-gnu",
    ],
    versions = [_RUSTC_VERSION],
)

load("@rules_rust//crate_universe:defs.bzl", "crates_repository")

crates_repository(
    name = "com_github_datadog_rules_oci_crate_index",
    cargo_lockfile = "//:Cargo.lock",
    lockfile = "//:cargo-bazel-lock.json",
    manifests = [
        "//:Cargo.toml",
        "//ocitool:Cargo.toml",
    ],
    rust_version = _RUSTC_VERSION,
)

load("@com_github_datadog_rules_oci_crate_index//:defs.bzl", "crate_repositories")

crate_repositories()
