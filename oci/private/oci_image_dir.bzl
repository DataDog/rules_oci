""" oci_image_dir """

load("@rules_pkg//pkg:mappings.bzl", "pkg_files")
load("@rules_pkg//pkg:pkg.bzl", "pkg_tar")
load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":common.bzl", "get_or_make_descriptor_file")
load(":oci_image_load.bzl", "oci_image_load")
load(":providers.bzl", "DebugInfo", "PlatformsInfo")

def oci_image_dir(
        *,
        image,  # str
        gzip = True,  # bool
        **kwargs):
    # -> None
    """Adds additional targets to oci_image and oci_image_index.

    Namely, creates targets for an OCI Image Layout directory and a tar file.

    Args:
        image: Name of the oci_image or oci_image_index target
        gzip: If true, creates a tar.gz file. If false, creates a tar file
        image: The label of an oci_image or oci_image_index
        **kwargs: Additional arguments to pass to the rule, e.g. tags or visibility
    """
    name_dir = "{}.dir".format(image)
    name_pkg_files = "{}.pkg_files".format(image)
    name_tar = "{}.tar".format(image)
    name_load = "{}.load".format(image)

    kwargs = dict(kwargs)

    # Ensure that the "manual" tag is always present
    tags = kwargs.pop("tags", None) or []
    tags = {k: True for k in tags}
    tags["manual"] = True
    tags = [k for k in tags.keys()]

    _oci_image_dir(
        name = name_dir,
        image = image,
        tags = tags,
        **kwargs
    )

    pkg_files_kwargs = dict(kwargs)
    pkg_files_kwargs["visibility"] = ["//visibility:private"]
    pkg_files(
        name = name_pkg_files,
        srcs = [name_dir],
        strip_prefix = ".",
        renames = {
            name_dir: "./",
        },
        tags = tags,
        **pkg_files_kwargs
    )

    if gzip:
        pkg_tar(
            name = name_tar,
            extension = "tar.gz",
            srcs = [name_pkg_files],
            package_file_name = "{}.tar.gz".format(image),
            strip_prefix = ".",
            tags = tags,
            **kwargs
        )
    else:
        pkg_tar(
            name = name_tar,
            srcs = [name_pkg_files],
            package_file_name = "{}.tar".format(image),
            strip_prefix = ".",
            tags = tags,
            **kwargs
        )

    oci_image_load(
        name = name_load,
        image = image,
        dir = name_dir,
        tar = name_tar,
        **kwargs
    )

def _impl(ctx):
    descriptor = get_or_make_descriptor_file(
        ctx,
        descriptor_provider = ctx.attr.image[OCIDescriptor],
        outpath = "{name}_/descriptor.json".format(name = ctx.label.name),
    )
    files = ctx.attr.image[OCILayout].files.to_list()

    out_dir = ctx.actions.declare_directory("{name}_/{name}".format(name = ctx.label.name))
    out_platforms = ctx.actions.declare_file("{name}_/platforms.json".format(name = ctx.label.name))

    args = ctx.actions.args()
    args.add("oci-dir")
    args.add("--descriptor-path", descriptor.path)
    args.add("--out-dir", out_dir.path)
    args.add("--out-platforms-path", out_platforms.path)
    for f in files:
        args.add("--file", f.path)

    env = {} if not ctx.attr._debug[DebugInfo].debug else {"RUST_LOG": "debug"}

    ctx.actions.run(
        arguments = [args],
        executable = ctx.executable._ocitool,
        inputs = [descriptor] + files,
        outputs = [out_dir, out_platforms],
        env = env,
    )

    return [
        DefaultInfo(
            files = depset([out_dir]),
        ),
        PlatformsInfo(
            platforms = out_platforms,
        ),
    ]

_oci_image_dir = rule(
    implementation = _impl,
    attrs = {
        "image": attr.label(providers = [OCIDescriptor]),
        "_debug": attr.label(default = "//oci:debug"),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//ocitool",
            executable = True,
        ),
    },
)
