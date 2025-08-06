""" release """

load("@rules_go//go:def.bzl", "go_binary")
load("@rules_pkg//pkg:pkg.bzl", "pkg_tar")
load(":with_platform.bzl", "with_platform")

DEFAULT_OSES = ["linux", "darwin"]
DEFAULT_ARCHS = ["amd64", "arm64"]

# buildifier: disable=function-docstring
def go_binary_multi(name, embed, oses = DEFAULT_OSES, archs = DEFAULT_ARCHS, **kwargs):
    if "goos" in kwargs or "goarch" in kwargs:
        fail("go_binary_multi does not allow goos or goarch in kwargs")

    go_binary(
        name = name,
        embed = embed,
        tags = ["manual"],
        **kwargs
    )

    all_binaries = []
    for os in oses:
        for arch in archs:
            bin_name = "{}-{}-{}".format(name, os, arch)
            with_platform(
                name = bin_name,
                arch = arch,
                os = os,
                src = name,
            )
            all_binaries.append(bin_name)

    all_bin_name = "{}.all".format(name)
    native.filegroup(
        name = all_bin_name,
        srcs = all_binaries,
    )

# buildifier: disable=function-docstring
def release_rules_oci(name, rules, binaries, **kwargs):
    top_name = "{}.top".format(name)
    pkg_tar(
        name = top_name,
        empty_files = ["BUILD.bazel"],
        package_dir = "",
    )

    rules_name = "{}.rules".format(name)
    pkg_tar(
        name = rules_name,
        srcs = rules,
        package_dir = "/oci",
    )

    binaries_name = "{}.bin".format(name)
    pkg_tar(
        name = binaries_name,
        srcs = binaries,
        mode = "0755",
        package_dir = "/bin",
    )

    pkg_tar(
        name = name,
        extension = "tar.gz",
        deps = [
            rules_name,
            binaries_name,
            top_name,
        ],
        **kwargs
    )
