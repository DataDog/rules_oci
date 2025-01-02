"""A rule to create a directory in OCI Image Layout format."""

load("@rules_pkg//pkg:mappings.bzl", "pkg_files")
load("@rules_pkg//pkg:pkg.bzl", "pkg_tar")
load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":debug_flag.bzl", "DebugInfo")

def oci_image_layout(
        *,
        name,
        image_index,
        gzip = True,
        **kwargs):
    """Creates targets for an OCI Image Layout directory and a tar file

    See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
    for the specification of the OCI Image Format directory.

    Args:
        name: A unique name for the rule
        image_index: An oci_image_index label
        gzip: If true, creates a tar.gz file. If false, creates a tar file
        **kwargs: Additional arguments to pass to the underlying rules, e.g.
          tags or visibility
    """
    _oci_image_layout(
        name = name,
        image_index = image_index,
        **kwargs
    )

    kwargs_copy = dict(kwargs)
    kwargs_copy.pop("visibility", None)
    pkg_files(
        name = "{}.pkg_files".format(name),
        srcs = [":{}".format(name)],
        strip_prefix = ".",
        renames = {
            ":{}".format(name): "./",
        },
        visibility = ["//visibility:private"],
        **kwargs_copy
    )

    if gzip:
        pkg_tar(
            name = "{}.tar".format(name),
            extension = "tar.gz",
            srcs = ["{}.pkg_files".format(name)],
            package_file_name = "{}.tar.gz".format(name),
            strip_prefix = ".",
            **kwargs
        )
    else:
        pkg_tar(
            name = "{}.tar".format(name),
            srcs = ["{}.pkg_files".format(name)],
            package_file_name = "{}.tar".format(name),
            strip_prefix = ".",
            **kwargs
        )

def _impl(ctx):
    layout = ctx.attr.image_index[OCILayout]

    # layout_files contains all available blobs for the image.
    layout_files = ",".join([p.path for p in layout.files.to_list()])

    descriptor = ctx.attr.image_index[OCIDescriptor]
    out_dir = ctx.actions.declare_directory(ctx.label.name)

    ctx.actions.run(
        executable = ctx.executable._ocitool,
        arguments = [
            "--layout={layout}".format(layout = layout.blob_index.path),
            "--debug={debug}".format(debug = str(ctx.attr._debug[DebugInfo].debug)),
            "create-oci-image-layout",
            # We need to use the directory one level above bazel-out for the
            # layout-relative directory. This is because the paths in
            # oci_image_index's index.layout.json are of the form:
            # "bazel-out/os_arch-fastbuild/bin/...". Unfortunately, bazel
            # provides no direct way to access this directory, so here we traverse
            # up 3 levels from the bin directory.
            "--layout-relative={root}".format(root = ctx.bin_dir.path + "/../../../"),
            "--desc={desc}".format(desc = descriptor.descriptor_file.path),
            "--layout-files={layout_files}".format(layout_files = layout_files),
            "--out-dir={out_dir}".format(out_dir = out_dir.path),
        ],
        inputs = depset(
            direct = ctx.files.image_index + [layout.blob_index],
            transitive = [layout.files],
        ),
        outputs = [out_dir],
    )

    return [
        DefaultInfo(files = depset([out_dir])),
    ]

_oci_image_layout = rule(
    implementation = _impl,
    attrs = {
        "image_index": attr.label(providers = [OCILayout]),
        "_debug": attr.label(
            default = "//oci:debug",
            providers = [DebugInfo],
        ),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//go/cmd/ocitool",
            executable = True,
        ),
    },
)
