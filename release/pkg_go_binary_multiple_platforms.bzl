""" pkg_go_binary_multiple_platforms """

load("@rules_go//go:def.bzl", _go_binary = "go_binary")
load("@rules_pkg//pkg:mappings.bzl", "pkg_attributes", "pkg_files")

_GOOSS = ["darwin", "linux"]
_GOARCHS = ["amd64", "arm64"]

# buildifier: disable=function-docstring
def pkg_go_binary_multiple_platforms(
        *,
        name,  # str
        go_binary,  # label
        mode,  # str
        prefix,  # str | None
        **kwargs):
    """ pkg_go_binary_multiple_platforms

    Creates a pkg_files target called <name> that includes multiple go binary
    executables (one for each platform) based off the provided go binary target.

    In other words, the following 4 exectuable will be included in the pkg_files
    target called <name>:
      - <name>-darwin-amd64
      - <name>-darwin-arm64
      - <name>-linux-amd64
      - <name>-linux-arm64

    Args:
        name:
            - The name of the pkg_files target to create
            - Also determines the names of the executables insde the pkg_files
              target, which will be of the form "<name>-<GOOS>-<GOARCH>".
        go_binary: The go_binary target to embed multiple versions of
        mode: The mode of the executables in pkg_files
        prefix: The prefix to pass to pkg_files
        **kwargs: Additional arguments to pass to pkg_files
    """
    all_binaries = []
    for goos in _GOOSS:
        for goarch in _GOARCHS:
            bin_name = "{}-{}-{}".format(name, goos, goarch)
            _go_binary(
                name = bin_name,
                embed = go_binary,
                goos = goos,
                goarch = goarch,
            )
            all_binaries.append(bin_name)

    pkg_files(
        name = name,
        srcs = all_binaries,
        attributes = pkg_attributes(
            mode = mode,
        ),
        prefix = prefix,
        **kwargs
    )
