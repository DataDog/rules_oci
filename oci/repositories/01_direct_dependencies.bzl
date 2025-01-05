""" direct dependencies """

load("@bazel_tools//tools/build_defs/repo:http.bzl", _http_archive = "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def http_archive(**kwargs):
    maybe(_http_archive, **kwargs)

def rules_oci_direct_dependencies():
    # 2024-05-08
    http_archive(
        name = "aspect_bazel_lib",
        sha256 = "87ab4ec479ebeb00d286266aca2068caeef1bb0b1765e8f71c7b6cfee6af4226",
        strip_prefix = "bazel-lib-2.7.3",
        url = "https://github.com/aspect-build/bazel-lib/releases/download/v2.7.3/bazel-lib-v2.7.3.tar.gz",
    )

    # 2024-08-01
    http_archive(
        name = "bazel_gazelle",
        sha256 = "8ad77552825b078a10ad960bec6ef77d2ff8ec70faef2fd038db713f410f5d87",
        urls = [
            "https://github.com/bazel-contrib/bazel-gazelle/releases/download/v0.38.0/bazel-gazelle-v0.38.0.tar.gz",
        ],
    )

    # 2024-04-25
    http_archive(
        name = "bazel_skylib",
        sha256 = "9f38886a40548c6e96c106b752f242130ee11aaa068a56ba7e56f4511f33e4f2",
        urls = [
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.6.1/bazel-skylib-1.6.1.tar.gz",
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.6.1/bazel-skylib-1.6.1.tar.gz",
        ],
    )

    # 2024-10-22
    http_archive(
        name = "com_google_protobuf",
        integrity = "sha256-fD69eq7dhvpdxHmg/agD9gLKr3jYr/fOg7ieG4rnRCo=",
        strip_prefix = "protobuf-28.3",
        urls = ["https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protobuf-28.3.tar.gz"],
    )

    # 2024-05-06
    http_archive(
        name = "io_bazel_rules_go",
        sha256 = "f74c98d6df55217a36859c74b460e774abc0410a47cc100d822be34d5f990f16",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.47.1/rules_go-v0.47.1.zip",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.47.1/rules_go-v0.47.1.zip",
        ],
    )

    # 2024-11-19
    http_archive(
        name = "rules_cc",
        urls = ["https://github.com/bazelbuild/rules_cc/releases/download/0.0.17/rules_cc-0.0.17.tar.gz"],
        sha256 = "abc605dd850f813bb37004b77db20106a19311a96b2da1c92b789da529d28fe1",
        strip_prefix = "rules_cc-0.0.17",
    )

    # 2024-02-08
    http_archive(
        name = "rules_pkg",
        urls = [
            "https://github.com/bazelbuild/rules_pkg/releases/download/0.10.1/rules_pkg-0.10.1.tar.gz",
        ],
        sha256 = "d250924a2ecc5176808fc4c25d5cf5e9e79e6346d79d5ab1c493e289e722d1d0",
    )

    # 2024-11-07
    http_archive(
        name = "rules_rust",
        integrity = "sha256-r09Wyq5QqZpov845sUG1Cd1oVIyCBLmKt6HK/JTVuwI=",
        urls = ["https://github.com/bazelbuild/rules_rust/releases/download/0.54.1/rules_rust-v0.54.1.tar.gz"],
    )
