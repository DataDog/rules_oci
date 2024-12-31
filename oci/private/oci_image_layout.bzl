"""A rule to create a directory in OCI Image Layout format."""

load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":debug_flag.bzl", "DebugInfo")

def _oci_image_layout_impl(ctx):
    toolchain = ctx.toolchains["//oci:toolchain"]

    layout = ctx.attr.manifest[OCILayout]

    # layout_files contains all available blobs for the image.
    layout_files = ",".join([p.path for p in layout.files.to_list()])

    descriptor = ctx.attr.manifest[OCIDescriptor]
    out_dir = ctx.actions.declare_directory(ctx.label.name)

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
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
        inputs =
            depset(
                direct = ctx.files.manifest + [layout.blob_index],
                transitive = [layout.files],
            ),
        outputs = [
            out_dir,
        ],
        use_default_shell_env = True,
    )

    return DefaultInfo(files = depset([out_dir]))

oci_image_layout = rule(
    doc = """
        Writes an OCI Image Index and related blobs to an OCI Image Format
        directory. See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
        for the specification of the OCI Image Format directory.

        All blobs must be provided in the manifest's OCILayout provider, in the
        files attribute. If blobs are missing, creation of the OCI Image Layout
        will fail.
    """,
    implementation = _oci_image_layout_impl,
    attrs = {
        "manifest": attr.label(
            doc = """
                An OCILayout index to be written to the OCI Image Format directory.
            """,
            providers = [OCILayout],
        ),
        "_debug": attr.label(
            default = "//oci:debug",
            providers = [DebugInfo],
        ),
    },
    provides = [
        DefaultInfo,
    ],
    toolchains = ["//oci:toolchain"],
)
