load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")
load("@com_datadoghq_cnab_tools//rules/oci:providers.bzl", "OCIDescriptor", "OCILayout", "OCITOOL_ATTR")

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
    descriptor_file = ctx.actions.declare_file("{}.descriptor.json".format(ctx.label.name))

    ctx.actions.run(
        executable = ctx.executable._ocitool,
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
    attrs = {
        "files": attr.label_list(),
        "directory": attr.string(),
        "symlinks": attr.string_dict(),
        "_ocitool": OCITOOL_ATTR,
    },
    outputs = {
        "layer": "%{name}-layer.tar",
    },
)

def _oci_image_index_impl(ctx):
    layout_files = depset(None, transitive = [m[OCILayout].files for m in ctx.attr.manifests])

    index_desc_file = ctx.actions.declare_file("{}.index.descriptor.json".format(ctx.label.name))
    index_file = ctx.actions.declare_file("{}.index.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.index.layout.json".format(ctx.label.name))

    desc_files = []
    for manifest in ctx.attr.manifests:
        desc_files.append(get_descriptor_file(ctx, manifest[OCIDescriptor]))

    ctx.actions.run(
        executable = ctx.executable._ocitool,
        arguments = ["--layout={}".format(m[OCILayout].blob_index.path) for m in ctx.attr.manifests] +
        [
            "--debug",
                        "create-index",
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
    attrs = {
        "manifests": attr.label_list(),
        "_ocitool": OCITOOL_ATTR,
    },
)

def _oci_image_impl(ctx):
    layout = ctx.attr.base[OCILayout]

    base_desc = get_descriptor_file(ctx, ctx.attr.base[OCIDescriptor])

    manifest_desc_file = ctx.actions.declare_file("{}.manifest.descriptor.json".format(ctx.label.name))
    manifest_file = ctx.actions.declare_file("{}.manifest.json".format(ctx.label.name))
    config_file = ctx.actions.declare_file("{}.config.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.layout.json".format(ctx.label.name))

    ctx.actions.run(
        executable = ctx.executable._ocitool,
        arguments = [
                        "--layout={}".format(layout.blob_index.path),
                        "append-layers",
                        "--base={}".format(base_desc.path),
                        "--os={}".format(ctx.attr.os),
                        "--arch={}".format(ctx.attr.arch),
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
    attrs = {
        "base": attr.label(
            mandatory = True,
            providers = [
                OCIDescriptor,
                OCILayout,
            ],
        ),
        "os": attr.string(),
        "arch": attr.string(),
        "layers": attr.label_list(),
        "_ocitool": OCITOOL_ATTR,
    },
)

def dd_go_image(name, base, archs, binary_name = "", binary_dir = "/app", **kwargs):
    os = "linux"

    visibility = kwargs.get("visibility", None)
    tags = kwargs.get("tags", None)

    if binary_name == "":
        binary_name = name

    manifests = []
    for arch in archs:
        go_binary_name = "{name}.{os}-{arch}-go-binary".format(name = name, os = os, arch = arch)
        go_binary_out = "{binary_name}-{os}-{arch}".format(binary_name = binary_name, os = os, arch = arch)
        go_binary(
            name = go_binary_name,
            goos = os,
            goarch = arch,
            out = go_binary_out,
            **kwargs
        )

        layer_name = "{name}.{os}-{arch}-go-layer".format(name = name, os = os, arch = arch)
        oci_image_layer(
            name = layer_name,
            files = [
                go_binary_name,
            ],
            symlinks = {
                "{}/{}".format(binary_dir, binary_name): "{}/{}".format(binary_dir, go_binary_out),
            },
            directory = binary_dir,
        )

        image_name = "{name}.{os}-{arch}-image".format(name = name, os = os, arch = arch)
        oci_image(
            name = image_name,
            base = base,
            layers = [
                layer_name,
            ],
            os = os,
            arch = arch,
        )
        manifests.append(image_name)

    oci_image_index(
        name = name,
        manifests = manifests,
        tags = tags,
        visibility = visibility,
    )
