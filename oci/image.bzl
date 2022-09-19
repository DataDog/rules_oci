load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load("//oci:ctx.bzl", "oci_ctx")

def get_descriptor_file(octx, desc):
    if hasattr(desc, "descriptor_file"):
        return desc.descriptor_file

    out = octx.actions.declare_file("{}.digest".format(octx.prefix))
    octx.actions.write(
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
                    ["--symlink={}={}".format(k, v) for k, v in ctx.attr.symlinks.items()] +
                    ["--file-map={}={}".format(k.files.to_list()[0].path, v) for k, v in ctx.attr.file_map.items()],
        inputs = ctx.files.files + ctx.files.file_map,
        outputs = [
            descriptor_file,
            ctx.outputs.layer,
        ],
    )

    return [
        OCIDescriptor(
            file = ctx.outputs.layer,
            descriptor_file = descriptor_file,
        ),
    ]

oci_image_layer = rule(
    implementation = _oci_image_layer_impl,
    doc = """
    """,
    provides = [OCIDescriptor],
    attrs = {
        "files": attr.label_list(
            doc = """

            """,
            allow_files = True,
        ),
        "directory": attr.string(
            doc = """
            """,
        ),
        "symlinks": attr.string_dict(
            doc = """
            """,
        ),
        "file_map": attr.label_keyed_string_dict(
            allow_files =  True,
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

    desc_files = []
    for manifest in ctx.attr.manifests:
        desc_files.append(get_descriptor_file(ctx, manifest[OCIDescriptor]))

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = ["--layout={}".format(m[OCILayout].blob_index.path) for m in ctx.attr.manifests] +
        [
            "--debug",
                        "create-index",
                        "--out-index={}".format(index_file.path),
                        "--out-layout={}".format(layout_file.path),
                        "--outd={}".format(index_desc_file.path),
                    ] +
                    ["--desc={}".format(d.path) for d in desc_files] +
                    ["--annotations={}={}".format(k, v) for k, v in ctx.attr.annotations.items()],
        inputs = desc_files + layout_files.to_list(),
        outputs = [
            index_file,
            index_desc_file,
            layout_file,
        ],
    )

    return [
        OCIDescriptor(
            descriptor_file = index_desc_file,
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

def create_oci_image_manifest(octx, layers, annotations={}, base=None, version_file=None, platform=None):
    base_desc = get_descriptor_file(ctx, ctx.attr.base[OCIDescriptor])

    manifest_desc_file = ctx.actions.declare_file("{}.manifest.descriptor.json".format(octx.prefix))
    manifest_file = ctx.actions.declare_file("{}.manifest.json".format(octx.prefix))
    config_file = ctx.actions.declare_file("{}.config.json".format(octx.prefix))
    layout_file = ctx.actions.declare_file("{}.layout.json".format(octx.prefix))

    args = []
    inputs = []

    if base != None:
        layout = ctx.attr.base[OCILayout]
        args.append("--layout={}".format(layout.blob_index.path))
        inputs += layout.files.to_list()
        inputs.append(layout.blob_index)

    args.append("append-layers")

    if version_file != None:
        args.append("--bazel-version-file={}".format(version_file.path))
        args.append(ctx.version_file)

    if platform != None:
        args.append("--os={}".format(ctx.attr.os))
        args.append("--arch={}".format(ctx.attr.arch))

    if base != None:
        base_desc := base[OCIDescriptor]
        args.append("--base={}".format(base_desc.path))
        inputs.append(base_desc.descriptor_file)

    args += [
        "--out-manifest={}".format(manifest_file.path),
        "--out-config={}".format(config_file.path),
        "--out-layout={}".format(layout_file.path),
        "--outd={}".format(manifest_desc_file.path),
    ]

    args += ["--layer={}".format(f[OCIDescriptor].file.path) for f in layers]
    inputs += [f[OCIDescriptor].file for f in layers]

    args += ["--annotations={}={}".format(k, v) for k, v in annotations.items()],

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = args,
        inputs = inputs,
        outputs = [
            manifest_file,
            config_file,
            layout_file,
            manifest_desc_file,
        ],
    )

    return struct(
        manifest_desc = OCIDescriptor(
            file = manifest_file,
            descriptor_file = manifest_desc_file,
        ),
        layout = OCILayout(
            blob_index = layout_file,
            files = depset(layers + [manifest_file, config_file, layout_file]),
        ),
    )

def create_oci_index_manifest(octx, manifests = [], annotations = {}):
    return struct(
        index_desc = OCIDescriptor(),
        layout = OCILayout(),
    )

def _oci_image_impl(ctx):
    octx := oci_ctx(ctx)

    rt = oci_image_manifest(
        octx,
        base = ctx.attr.base,
        layers = ctx.attr.layers,
        platform = OCIPlatform(
            os = ctx.attr.os,
            arch = ctx.attr.arch,
        ),
        annotations = ctx.attr.annotations,
    )

    return [
        rt.manifest_desc,
        rt.layout,
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
            providers = [
                OCIDescriptor,
            ],
        ),
        "annotations": attr.string_dict(
            doc = """
            """,
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)
