load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout", "OCIReferenceInfo")
load("@com_github_datadog_rules_oci//oci:debug_flag.bzl", "DebugInfo")

def _oci_push_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout = ctx.attr.manifest[OCILayout]

    ref = "{registry}/{repository}".format(
        registry = ctx.attr.registry,
        repository = ctx.attr.repository,
    )

    tag = ctx.expand_make_variables("tag", ctx.attr.tag, {})
    if tag:
        ref = "{ref}:{tag}".format(
            ref = ref,
            tag = tag,
        )

    digest_file = ctx.actions.declare_file("{name}.digest".format(name = ctx.label.name))
    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
            "digest",
            "--desc={desc}".format(desc = ctx.attr.manifest[OCIDescriptor].descriptor_file.path),
            "--out={out}".format(out = digest_file.path),
        ],
        inputs = [
            ctx.attr.manifest[OCIDescriptor].descriptor_file,
        ],
        outputs = [
            digest_file,
        ],
    )

    headers = ""
    for k, v in ctx.attr.headers.items():
        headers = headers + " --headers={}={}".format(k, v)

    ctx.actions.write(
        content = """
        {tool}  \\
        --layout {layout} \\
        --debug={debug} \\
        push \\
        --layout-relative {root} \\
        --desc {desc} \\
        --target-ref {ref} \\
        {headers} \\
        """.format(
            root = ctx.bin_dir.path,
            tool = toolchain.sdk.ocitool.short_path,
            layout = layout.blob_index.short_path,
            desc = ctx.attr.manifest[OCIDescriptor].descriptor_file.short_path,
            ref = ref,
            debug = str(ctx.attr._debug[DebugInfo].debug),
            headers = headers,
        ),
        output = ctx.outputs.executable,
        is_executable = True,
    )

    return [
        DefaultInfo(
            runfiles = ctx.runfiles(
                files = layout.files.to_list() +
                        [toolchain.sdk.ocitool, ctx.attr.manifest[OCIDescriptor].descriptor_file, layout.blob_index],
            ),
        ),
        OCIReferenceInfo(
            registry = ctx.attr.registry,
            repository = ctx.attr.repository,
            digest = digest_file,
        ),
    ]

oci_push = rule(
    doc = """
        Pushes a manifest or a list of manifests to an OCI registry.
    """,
    implementation = _oci_push_impl,
    executable = True,
    attrs = {
        "manifest": attr.label(
            doc = """
                A manifest to push to a registry. If an OCILayout index, then
                push all artifacts with a 'org.opencontainers.image.ref.name'
                annotation.
            """,
            providers = [OCILayout],
        ),
        "registry": attr.string(
            doc = """
                A registry host to push to, if not present consult the toolchain.
            """,
        ),
        "repository": attr.string(
            doc = """
                A repository to push to, if not present consult the toolchain.
            """,
        ),
        "tag": attr.string(
            doc = """
                (optional) A tag to include in the target reference."
            """,
        ),
        "headers": attr.string_dict(
            doc = """
                (optional) A list of key/values to include with the push.
            """,
        ),
        "_debug": attr.label(
            default = "//oci:debug",
            providers = [DebugInfo],
        ),
    },
    provides = [
        OCIReferenceInfo,
    ],
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)
