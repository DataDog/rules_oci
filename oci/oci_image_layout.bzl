load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load("@com_github_datadog_rules_oci//oci:debug_flag.bzl", "DebugInfo")

def _oci_image_layout_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout = ctx.attr.manifest[OCILayout]
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
            "--registry={registry}".format(registry = ctx.attr.registry),
            "--repository={repository}".format(repository = ctx.attr.repository),
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
        --incompatible_strict_action_env=false
        --spawn_strategy=local

        The incompatible_strict_action_env flag is required because in order to
        access the registry, a credential helper executable (named
        docker-credential-<SOMETHING>) must be available so that ocitool can
        execute it. The incompatible_strict_action_env flag makes the system
        PATH available to bazel rules.

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
            providers = [OCILayout],
        ),
        "registry": attr.string(
            doc = """
                A registry host that contains images referenced by the OCILayout index,
                if not present consult the toolchain.
            """,
        ),
        "repository": attr.string(
            doc = """
                A repository that contains images referenced by the OCILayout index,
                if not present consult the toolchain.
            """,
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
