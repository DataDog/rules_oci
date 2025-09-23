""" image """

load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")

def oci_image(
        name,
        base,
        annotations = None,
        arch = None,
        entrypoint = None,
        env = None,
        labels = None,
        layers = None,
        os = None,
        tars = None,
        **kwargs):
    """ oci_image

    Creates a new image manifest and config by appending the `layers` to an
    existing image manifest and config defined by `base`.  If `base` is an image
    index, then `os` and `arch` will be used to extract the image manifest.

    Args:
        name: The name of the rule.
        base: A base image, as defined by oci_pull or oci_image.
        annotations: OCI Annotations to add to the manifest.
        arch: Used to extract a manifest from base if base is an index.
        entrypoint: A list of entrypoints for the image; these will be inserted
            into the generated container configuration.
        env: Entries are in the format of `VARNAME=VARVALUE`. These values act
            as defaults and are merged with any specified when creating a
            container.
        labels: Labels that will be applied to the image configuration, as
            defined in the OCI config. These behave the same way as docker
            LABEL. In particular, labels from the base image are inherited. An
            empty value for a label will cause that label to be deleted. For
            backwards compatibility, if this is not set, then the value of
            annotations will be used instead.
        layers: A list of layers defined by oci_image_layer.
        os: Used to extract a manifest from base if base is an index.
        tars: A list of tars to add as layers.
        **kwargs: Additional keyword arguments, e.g. tags or visibility
    """
    if entrypoint == None:
        entrypoint_override = False
    else:
        entrypoint_override = True

    _oci_image(
        name = name,
        base = base,
        annotations = annotations,
        arch = arch,
        entrypoint = entrypoint,
        entrypoint_override = entrypoint_override,
        env = env,
        labels = labels,
        layers = layers,
        os = os,
        tars = tars,
        **kwargs
    )

# buildifier: disable=function-docstring
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

def _oci_image_index_impl(ctx):
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    layout_files = depset(None, transitive = [m[OCILayout].files for m in ctx.attr.manifests])

    index_desc_file = ctx.actions.declare_file("{}.index.descriptor.json".format(ctx.label.name))
    index_file = ctx.actions.declare_file("{}.index.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.index.layout.json".format(ctx.label.name))

    desc_files = []
    for manifest in ctx.attr.manifests:
        desc_files.append(get_descriptor_file(ctx, manifest[OCIDescriptor]))

    outputs = [
        index_file,
        index_desc_file,
        layout_file,
    ]

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = ["--layout={}".format(m[OCILayout].blob_index.path) for m in ctx.attr.manifests] +
                    [
                        "create-index",
                        "--out-index={}".format(index_file.path),
                        "--out-layout={}".format(layout_file.path),
                        "--outd={}".format(index_desc_file.path),
                    ] +
                    ["--desc={}".format(d.path) for d in desc_files] +
                    ["--annotations={}={}".format(k, v) for k, v in ctx.attr.annotations.items()],
        mnemonic = "OCImageCreateIndex",
        inputs = desc_files + layout_files.to_list(),
        outputs = outputs,
    )

    return [
        OCIDescriptor(
            descriptor_file = index_desc_file,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(direct = [index_file, layout_file], transitive = [layout_files]),
        ),
        DefaultInfo(
            files = depset(outputs),
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

    base_desc = get_descriptor_file(ctx, ctx.attr.base[OCIDescriptor])
    base_layout = ctx.attr.base[OCILayout]

    manifest_desc_file = ctx.actions.declare_file("{}.manifest.descriptor.json".format(ctx.label.name))
    manifest_file = ctx.actions.declare_file("{}.manifest.json".format(ctx.label.name))
    config_file = ctx.actions.declare_file("{}.config.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.layout.json".format(ctx.label.name))

    annotations = ctx.attr.annotations

    # Backwards compatibility: code that doesn't use the labels attr will expect annotations to be
    # used as labels
    labels = ctx.attr.labels or ctx.attr.annotations

    layer_descriptor_files = [get_descriptor_file(ctx, f[OCIDescriptor]) for f in ctx.attr.layers]
    layer_and_descriptor_paths = zip(
        [f.path for f in ctx.files.layers],
        [f.path for f in layer_descriptor_files],
    )

    tars = []
    for tar in ctx.attr.tars:
        tmp = tar.files.to_list()
        if len(tmp) != 1:
            fail("tar must contain exactly one file")
        tar = tmp[0]
        tars.append(tar)

    stamp_args = []
    if maybe_stamp(ctx):
        stamp_args.append("--bazel-version-file={}".format(ctx.version_file.path))

    arguments = [
        "--layout={}".format(base_layout.blob_index.path),
        "append-layers",
        "--base={}".format(base_desc.path),
        "--os={}".format(ctx.attr.os),
        "--arch={}".format(ctx.attr.arch),
        "--out-manifest={}".format(manifest_file.path),
        "--out-config={}".format(config_file.path),
        "--out-layout={}".format(layout_file.path),
        "--outd={}".format(manifest_desc_file.path),
    ] + [
        "--layer={}={}".format(layer, descriptor)
        for layer, descriptor in layer_and_descriptor_paths
    ] + [
        "--tar={}".format(tar.path)
        for tar in tars
    ] + [
        "--annotations={}={}".format(k, v)
        for k, v in annotations.items()
    ] + [
        "--labels={}={}".format(k, v)
        for k, v in labels.items()
    ] + [
        "--env={}".format(env)
        for env in ctx.attr.env
    ] + stamp_args

    default_info_files = [
        config_file,
        layout_file,
        manifest_desc_file,
        manifest_file,
    ]

    inputs = [
        ctx.version_file,
        base_desc,
        base_layout.blob_index,
    ] + ctx.files.layers + layer_descriptor_files + base_layout.files.to_list() + tars

    if ctx.attr.entrypoint_override:
        entrypoint_config_file = ctx.actions.declare_file("{}.entrypoint.config.json".format(ctx.label.name))
        entrypoint_config = struct(
            entrypoint = ctx.attr.entrypoint,
        )
        ctx.actions.write(
            output = entrypoint_config_file,
            content = json.encode(entrypoint_config),
        )
        arguments.append("--entrypoint={}".format(entrypoint_config_file.path))
        default_info_files.append(entrypoint_config_file)
        inputs.append(entrypoint_config_file)

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = arguments,
        inputs = inputs,
        mnemonic = "OCIImageAppendLayers",
        outputs = [
            config_file,
            layout_file,
            manifest_desc_file,
            manifest_file,
        ],
    )

    return [
        OCIDescriptor(
            descriptor_file = manifest_desc_file,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(
                ctx.files.layers + ctx.files.tars + [
                    manifest_file,
                    config_file,
                    layout_file,
                ],
                transitive = [base_layout.files],
            ),
        ),
        DefaultInfo(
            files = depset(default_info_files),
        ),
    ]

_oci_image = rule(
    implementation = _oci_image_impl,
    attrs = dict({
        "annotations": attr.string_dict(),
        "arch": attr.string(),
        "base": attr.label(
            mandatory = True,
            providers = [OCIDescriptor, OCILayout],
        ),
        "entrypoint": attr.string_list(),
        "entrypoint_override": attr.bool(),
        "env": attr.string_list(),
        "labels": attr.string_dict(),
        "layers": attr.label_list(
            providers = [
                OCIDescriptor,
            ],
        ),
        "os": attr.string(),
        "tars": attr.label_list(
            allow_files = [".tar", ".tar.gz", ".tgz", ".tar.zst"],
        ),
    }, **STAMP_ATTRS),
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
    provides = [OCIDescriptor, OCILayout],
)
