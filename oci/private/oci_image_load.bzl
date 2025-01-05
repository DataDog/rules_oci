""" oci_image_load """

load("@aspect_bazel_lib//lib:paths.bzl", "BASH_RLOCATION_FUNCTION", "to_rlocation_path")
load("//oci:providers.bzl", "OCIDescriptor")
load(":providers.bzl", "PlatformsInfo")

def oci_image_load(
        name,
        dir,
        image,
        tar,
        **kwargs):
    """Creates an executable target that loads and OCI image into docker

    Args:
        name: Name of the target
        dir: The label of the oci_dir target associated with the image
        image: The label of the oci_image or oci_image_index target to load
        tar: The label of the tar file associated with the image
        **kwargs: Additional arguments to pass to the rule, e.g. tags or visibility
    """
    kwargs = dict(kwargs)

    # Ensure that the "manual" tag is always present
    tags = kwargs.pop("tags", None) or []
    tags = {k: True for k in tags}
    tags["manual"] = True
    tags = [k for k in tags.keys()]

    _oci_image_load(
        name = name,
        dir = dir,
        image = image,
        tar = tar,
        tags = tags,
        **kwargs
    )

def _impl(ctx):
    platforms = ctx.attr.dir[PlatformsInfo].platforms
    ocitool = ctx.executable._ocitool

    repository = "bazel/{}/{}".format(
        ctx.attr.image.label.package,
        ctx.attr.image.label.name,
    )

    exe = ctx.actions.declare_file("{}_/run.sh".format(ctx.label.name))
    ctx.actions.write(
        output = exe,
        content = """
#!/usr/bin/env bash
set -euo pipefail

{BASH_RLOCATION_FUNCTION}

ocitool="$(rlocation "{ocitool}")"
platforms="$(rlocation "{platforms}")"
tar="$(rlocation "{tar}")"

"${{ocitool}}" \\
    oci-load \\
    --platforms-path "${{platforms}}" \\
    --repository "{repository}" \\
    --tar-path "${{tar}}"
""".strip().format(
            BASH_RLOCATION_FUNCTION = BASH_RLOCATION_FUNCTION,
            ocitool = to_rlocation_path(ctx, ocitool),
            platforms = to_rlocation_path(ctx, platforms),
            repository = repository,
            tar = to_rlocation_path(ctx, ctx.file.tar),
        ),
    )

    runfiles = ctx.runfiles(files = [ctx.file.tar, ocitool, platforms])
    runfiles = runfiles.merge(ctx.attr._bash_runfiles.default_runfiles)
    runfiles = runfiles.merge(ctx.attr._ocitool.default_runfiles)

    return [
        DefaultInfo(
            files = depset([exe]),
            runfiles = runfiles,
            executable = exe,
        ),
    ]

_oci_image_load = rule(
    implementation = _impl,
    attrs = {
        "dir": attr.label(
            providers = [PlatformsInfo],
            mandatory = True,
        ),
        "image": attr.label(providers = [OCIDescriptor]),
        "tar": attr.label(
            allow_single_file = True,
            mandatory = True,
        ),
        "_bash_runfiles": attr.label(default = "@bazel_tools//tools/bash/runfiles"),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//ocitool",
            executable = True,
        ),
    },
    executable = True,
)
