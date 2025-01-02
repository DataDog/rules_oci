""" oci_image_layer """

load("//oci:providers.bzl", "OCIDescriptor")

def oci_image_layer(
        *,
        name,
        directory = "/",  # str
        file_map = None,  # dict[label, str]
        files = None,  # list[label]
        symlinks = None,  # dict[str, str]
        **kwargs):
    # -> None
    """ oci_image_layer

    Args:
        name: A unique name for this rule
        directory: Directory in the tarball to place the `files`
        file_map: Dictionary of file -> file location in tarball
        files: List of files to include under `directory`
        symlinks: Dictionary of symlink -> target entries to place in the tarball
        **kwargs: Additional arguments to pass to the rule, e.g. `tags` or `visibility`
    """
    file_map = file_map or {}
    files = files or []
    symlinks = symlinks or {}

    if len(files) == 0 and len(file_map) == 0:
        fail("At least one of `files` or `file_map` must be provided")

    _oci_image_layer(
        name = name,
        directory = directory,
        file_map = file_map,
        files = files,
        symlinks = symlinks,
        **kwargs
    )

def _impl(ctx):
    toolchain = ctx.toolchains["//oci:toolchain"]

    descriptor_file = ctx.actions.declare_file("{}.descriptor.json".format(ctx.label.name))

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
                        "create-layer",
                        "--out={}".format(ctx.outputs.layer.path),
                        "--outd={}".format(descriptor_file.path),
                        "--dir={}".format(ctx.attr.directory),
                        "--bazel-label={}".format(ctx.label),
                    ] +
                    ["--file={}".format(f.path) for f in ctx.files.files] +
                    ["--symlink={}={}".format(k, v) for k, v in ctx.attr.symlinks.items()] +
                    ["--file-map={}={}".format(k.files.to_list()[0].path, v) for k, v in ctx.attr.file_map.items()],
        inputs = ctx.files.files + ctx.files.file_map,
        outputs = [
            descriptor_file,
            ctx.outputs.layer,
        ],
    )

    return [
        DefaultInfo(
            files = depset([ctx.outputs.layer, descriptor_file]),
        ),
        OCIDescriptor(
            descriptor_file = descriptor_file,
            file = ctx.outputs.layer,
        ),
    ]

_oci_image_layer = rule(
    implementation = _impl,
    attrs = {
        "files": attr.label_list(allow_files = True),
        "directory": attr.string(),
        "symlinks": attr.string_dict(),
        "file_map": attr.label_keyed_string_dict(allow_files = True),
    },
    toolchains = ["//oci:toolchain"],
    outputs = {
        "layer": "%{name}-layer.tar.gz",
    },
)
