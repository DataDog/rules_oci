load("@rules_pkg//pkg:pkg.bzl", "pkg_tar")
load("@rules_go//go:def.bzl", "go_binary")

DEFAULT_GOOSS = ["linux", "darwin"]
DEFAULT_GOARCHS = ["amd64", "arm64"]

def go_binary_multi(name, embed, gooss = DEFAULT_GOOSS, goarchs = DEFAULT_GOARCHS, **kwargs):
    if "goos" in kwargs or "goarch" in kwargs:
        fail("go_binary_multi does not allow goos or goarch in kwargs")

    go_binary(
        name = name,
        embed = embed,
        **kwargs
    )

    all_binaries = []
    for goos in gooss:
        for goarch in goarchs:
            bin_name = "{}-{}-{}".format(name, goos, goarch)
            go_binary(
                name = bin_name,
                embed = embed,
                goos = goos,
                goarch = goarch,
                tags = ["manual"],
                **kwargs
            )
            all_binaries.append(bin_name)

    all_bin_name = "{}.all".format(name)
    native.filegroup(
        name = all_bin_name,
        srcs = all_binaries,
    )

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
