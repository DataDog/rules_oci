""" oci_image_layer """

load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor")

def oci_image_layer(
        *,
        name,
        directory = None,  # str | None
        files = None,  # list[str] | None
        file_map = None,  # dict[label, str] | None,
        mode_map = None,  # dict[str, int] | None,
        owner_map = None,  # dict[str, str] | None,
        symlinks = None,  # dict[str, str] | None,
        **kwargs):
    """ Creates a tarball and an OCI descriptor for it

    Args:
        name: A unique name for this rule
        directory: Directory in the tarball to place the `files`
        files: List of files to include under `directory`
        file_map: Dictionary of file -> file location in tarball
        mode_map: Dictionary of file location in tarball -> mode int (e.g. 0o755)
        owner_map: Dictionary of file location in tarball -> owner:group string (e.g. '501:501')
        symlinks: Dictionary of symlink -> target entries to place in the tarball
        **kwargs: Additional arguments to pass to the rule, e.g. tags or visibility
    """
    mode_map = {k: str(v) for k, v in mode_map.items()} if mode_map else {}
    owner_map = owner_map or {}
    _oci_image_layer(
        name = name,
        directory = directory,
        files = files,
        file_map = file_map,
        mode_map = mode_map,
        owner_map = owner_map,
        symlinks = symlinks,
        **kwargs
    )

def _impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    descriptor_file = ctx.actions.declare_file("{}.descriptor.json".format(ctx.label.name))

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
                        "create-layer",
                        "--bazel-label={}".format(ctx.label),
                        "--dir={}".format(ctx.attr.directory),
                        "--out={}".format(ctx.outputs.layer.path),
                        "--outd={}".format(descriptor_file.path),
                    ] +
                    ["--file-map={}={}".format(k.files.to_list()[0].path, v) for k, v in ctx.attr.file_map.items()] +
                    ["--file={}".format(f.path) for f in ctx.files.files] +
                    ["--mode-map={}={}".format(k, v) for k, v in ctx.attr.mode_map.items()] +
                    ["--owner-map={}={}".format(k, v) for k, v in ctx.attr.owner_map.items()] +
                    ["--symlink={}={}".format(k, v) for k, v in ctx.attr.symlinks.items()],
        inputs = ctx.files.files + ctx.files.file_map,
        mnemonic = "OCIImageCreateLayer",
        outputs = [
            descriptor_file,
            ctx.outputs.layer,
        ],
    )

    return [
        OCIDescriptor(
            descriptor_file = descriptor_file,
            file = ctx.outputs.layer,
        ),
    ]

_oci_image_layer = rule(
    implementation = _impl,
    doc = "Create a tarball and an OCI descriptor for it",
    attrs = {
        "directory": attr.string(),
        "file_map": attr.label_keyed_string_dict(allow_files = True),
        "files": attr.label_list(allow_files = True),
        "mode_map": attr.string_dict(),
        "owner_map": attr.string_dict(),
        "symlinks": attr.string_dict(),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
    outputs = {
        "layer": "%{name}-layer.tar.gz",
    },
)
