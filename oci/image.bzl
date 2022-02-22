load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")

def get_descriptor_file(ctx, desc):
    if hasattr(desc, "descriptor_file"):
        return desc.descriptor_file

    out = ctx.actions.declare_file(desc.digest)
    ctx.actions.write(
        output = out,
        content = json.encode({
            "mediaType": desc.media_type,
            "size": desc.size,
            "digest": desc.digest,
            "urls": desc.urls,
            "annotations": desc.annotations,
        }),
    )

    return out

def _oci_image_layer_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    descriptor_file = ctx.actions.declare_file("{}.descriptor.json".format(ctx.label.name))

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
                        "create-layer",
                        "--out={}".format(ctx.outputs.layer.path),
                        "--outd={}".format(descriptor_file.path),
                        "--dir={}".format(ctx.attr.directory),
                    ] +
                    ["--file={}".format(f.path) for f in ctx.files.files] +
                    ["--symlink={}={}".format(k, v) for k, v in ctx.attr.symlinks.items()],
        inputs = ctx.files.files,
        outputs = [
            descriptor_file,
            ctx.outputs.layer,
        ],
    )

    return [
        OCIDescriptor(
            descriptor_file = descriptor_file,
        ),
    ]

oci_image_layer = rule(
    implementation = _oci_image_layer_impl,
    doc = """
    """,
    attrs = {
        "files": attr.label_list(
            doc = """

            """,
        ),
        "directory": attr.string(
            doc = """
            """,
        ),
        "symlinks": attr.string_dict(
            doc = """
            """,
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
    outputs = {
        "layer": "%{name}-layer.tar",
    },
)

def _oci_image_index_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout_files = depset(None, transitive = [m[OCILayout].files for m in ctx.attr.manifests])

    index_desc_file = ctx.actions.declare_file("{}.index.descriptor.json".format(ctx.label.name))
    index_file = ctx.actions.declare_file("{}.index.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.index.layout.json".format(ctx.label.name))

    annotation_strs = ["{}={}".format(key, value) for key, value in ctx.attr.annoations]

    desc_files = []
    for manifest in ctx.attr.manifests:
        desc_files.append(get_descriptor_file(ctx, manifest[OCIDescriptor]))

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = ["--layout={}".format(m[OCILayout].blob_index.path) for m in ctx.attr.manifests] +
        [
            "--debug",
                        "create-index",
                        "--annotations={}".format(",".join(annotation_strs)),
                        "--out-index={}".format(index_file.path),
                        "--out-layout={}".format(layout_file.path),
                        "--outd={}".format(index_desc_file.path),
                    ] +
                    ["--desc={}".format(d.path) for d in desc_files],
        inputs = desc_files + layout_files.to_list(),
        outputs = [
            index_file,
            index_desc_file,
            layout_file,
        ],
    )

    return [
        OCIDescriptor(
            file = index_desc_file,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(direct = [index_file, layout_file], transitive = [layout_files]),
        ),
    ]

oci_image_index = rule(
    implementation = _oci_image_index_impl,
    doc = """
    """,
    attrs = {
        "manifests": attr.label_list(
            doc = """
            """,
        ),
        "annotations": attr.string_dict(
            doc = """
            """,
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)

def _oci_image_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout = ctx.attr.base[OCILayout]

    base_desc = get_descriptor_file(ctx, ctx.attr.base[OCIDescriptor])

    manifest_desc_file = ctx.actions.declare_file("{}.manifest.descriptor.json".format(ctx.label.name))
    manifest_file = ctx.actions.declare_file("{}.manifest.json".format(ctx.label.name))
    config_file = ctx.actions.declare_file("{}.config.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.layout.json".format(ctx.label.name))

    annotation_strs = ["{}={}".format(key, value) for key, value in ctx.attr.annoations]

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
                        "--layout={}".format(layout.blob_index.path),
                        "append-layers",
                        "--base={}".format(base_desc.path),
                        "--os={}".format(ctx.attr.os),
                        "--arch={}".format(ctx.attr.arch),
                        "--annoations={}".format(",".join(annotation_strs)),
                        "--out-manifest={}".format(manifest_file.path),
                        "--out-config={}".format(config_file.path),
                        "--out-layout={}".format(layout_file.path),
                        "--outd={}".format(manifest_desc_file.path),
                    ] +
                    ["--layer={}".format(f.path) for f in ctx.files.layers],
        inputs = [base_desc, layout.blob_index] + ctx.files.layers + layout.files.to_list(),
        outputs = [
            manifest_file,
            config_file,
            layout_file,
            manifest_desc_file,
        ],
    )

    return [
        OCIDescriptor(
            descriptor_file = manifest_desc_file,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(ctx.files.layers + [manifest_file, config_file, layout_file]),
        ),
    ]

oci_image = rule(
    implementation = _oci_image_impl,
    doc = """
    """,
    attrs = {
        "base": attr.label(
            doc = """
            """,
            mandatory = True,
            providers = [
                OCIDescriptor,
                OCILayout,
            ],
        ),
        "os": attr.string(
            doc = """
            """,
        ),
        "arch": attr.string(
            doc = """
            """,
        ),
        "layers": attr.label_list(
            doc = """
            """,
        ),
        "annotations": attr.string_dict(
            doc = """
            """,
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)
