load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCIImageLayoutInfo", "OCILayout")
load("@com_github_datadog_rules_oci//oci:debug_flag.bzl", "DebugInfo")

def _oci_image_layout_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout = ctx.attr.manifest[OCILayout]
    base_layouts = ctx.attr.manifest[OCIImageLayoutInfo]
    base_layout_dirs = ""
    if base_layouts != None:
        base_layouts = base_layouts.oci_image_layout_dirs.to_list()
        base_layout_dirs = ",".join([p.path for p in base_layouts])


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
            "--layout-relative={root}".format(root = ctx.bin_dir.path+"/../../../"),
            "--desc={desc}".format(desc = descriptor.descriptor_file.path),
            "--base-image-layouts={base_layouts}".format(base_layouts = base_layout_dirs),
            "--out-dir={out_dir}".format(out_dir = out_dir.path),
        ],
        inputs = ctx.files.manifest,
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
        for the specification of the OCI Image Format directory. Local blobs are
        used where available, and if a referenced blob is not present, it is
        fetched from the provided OCI repository and placed in the output.

        In order for this rule to work correctly in its current state, the
        following flags must be provided to bazel:
        --spawn_strategy=local

        The spawn_strategy flag must be set to local because currently,
        oci_image_index is only declaring the new JSON files it creates as
        outputs; it's not declaring any manifests or layers from the images as
        outputs. By default, Bazel only permits rules to access specifically
        declared outputs of the rule's direct dependencies. In order for this
        rule to access the transitive set of outputs of all dependencies, we
        must disable bazel's sandboxing by setting spawn_strategy=local.
    """,
    # TODO(kim.mason): Fix oci_image/oci_image_index so they explicitly declare
    # outputs that include everything needed to build the image.
    # TODO(kim.mason): Make it so that Docker credential helpers are available
    # to oci_image_layout without making the system PATH available.
    implementation = _oci_image_layout_impl,
    attrs = {
        "manifest": attr.label(
            doc = """
                An OCILayout index to be written to the OCI Image Format directory.
            """,
            providers = [OCILayout, OCIImageLayoutInfo],
        ),
        "_debug": attr.label(
            default = "//oci:debug",
            providers = [DebugInfo],
        ),
    },
    provides = [
        DefaultInfo,
    ],
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)