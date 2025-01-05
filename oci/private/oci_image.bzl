""" oci_image """

load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":common.bzl", "MEDIA_TYPE_OCI_MANIFEST", "get_or_make_descriptor_file")
load(":oci_image_dir.bzl", "oci_image_dir")

def oci_image(
        name,
        base,  # label
        annotations = None,  # dict[str, str] | None
        arch = None,  # str | None
        entrypoint = None,  # list[str] | None
        env = None,  # dict[str, str] | None
        gzip = True,  # bool
        labels = None,  # dict[str, str] | None
        layers = None,  # list[label] | None
        os = None,  # str | None
        **kwargs):
    """Creates a "single-arch"" OCI image

    Also creates targets for an OCI Layout directory and a .tar.gz file

    Args:
        name: A unique name for the rule
        base: A label of an oci_image or oci_image_index
        annotations: A dictionary of annotations to add to the image
        arch: The architecture of the image
        entrypoint: A list of entrypoints for the image
        env: A list of environment variables to add to the image
        gzip: If true, creates a tar.gz file. If false, creates a tar file
        labels: A dictionary of labels to add to the image
        layers: A list of oci_image_layer labels
        os: The operating system of the image
        **kwargs: Additional arguments to pass to the underlying rules, e.g.
          tags or visibility
    """
    _oci_image(
        name = name,
        base = base,
        annotations = annotations,
        arch = arch,
        entrypoint = entrypoint,
        env = env,
        labels = labels,
        layers = layers,
        os = os,
        **kwargs
    )

    oci_image_dir(
        image = name,
        gzip = gzip,
        **kwargs
    )

def _impl(ctx):
    base_desc = get_or_make_descriptor_file(
        ctx,
        descriptor_provider = ctx.attr.base[OCIDescriptor],
    )
    base_layout = ctx.attr.base[OCILayout]

    manifest_desc_file = ctx.actions.declare_file("{}.manifest.descriptor.json".format(ctx.label.name))
    manifest_file = ctx.actions.declare_file("{}.manifest.json".format(ctx.label.name))
    config_file = ctx.actions.declare_file("{}.config.json".format(ctx.label.name))
    layout_file = ctx.actions.declare_file("{}.layout.json".format(ctx.label.name))

    entrypoint_config_file = ctx.actions.declare_file("{}.entrypoint.config.json".format(ctx.label.name))
    entrypoint_config = struct(
        entrypoint = ctx.attr.entrypoint,
    )

    ctx.actions.write(
        output = entrypoint_config_file,
        content = json.encode(entrypoint_config),
    )

    annotations = ctx.attr.annotations

    # Backwards compatibility: code that doesn't use the labels attr will expect annotations to be
    # used as labels
    labels = ctx.attr.labels or ctx.attr.annotations

    layer_descriptor_files = [
        get_or_make_descriptor_file(
            ctx,
            descriptor_provider = f[OCIDescriptor],
        )
        for f in ctx.attr.layers
    ]
    layer_and_descriptor_paths = zip(
        [f.path for f in ctx.files.layers],
        [f.path for f in layer_descriptor_files],
    )

    ctx.actions.run(
        executable = ctx.executable._ocitool,
        arguments = [
                        "--layout={}".format(base_layout.blob_index.path),
                        "append-layers",
                        "--bazel-version-file={}".format(ctx.version_file.path),
                        "--base={}".format(base_desc.path),
                        "--os={}".format(ctx.attr.os),
                        "--arch={}".format(ctx.attr.arch),
                        "--out-manifest={}".format(manifest_file.path),
                        "--out-config={}".format(config_file.path),
                        "--out-layout={}".format(layout_file.path),
                        "--outd={}".format(manifest_desc_file.path),
                        "--entrypoint={}".format(entrypoint_config_file.path),
                    ] +
                    [
                        "--layer={}={}".format(layer, descriptor)
                        for layer, descriptor in layer_and_descriptor_paths
                    ] +
                    ["--annotations={}={}".format(k, v) for k, v in annotations.items()] +
                    ["--labels={}={}".format(k, v) for k, v in labels.items()] +
                    ["--env={}".format(env) for env in ctx.attr.env],
        inputs = [
                     ctx.version_file,
                     base_desc,
                     base_layout.blob_index,
                     entrypoint_config_file,
                 ] + ctx.files.layers +
                 layer_descriptor_files +
                 base_layout.files.to_list(),
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
            media_type = MEDIA_TYPE_OCI_MANIFEST,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(
                ctx.files.layers + [manifest_file, config_file, layout_file],
                transitive = [base_layout.files],
            ),
        ),
        DefaultInfo(
            files = depset([
                entrypoint_config_file,
                manifest_file,
                config_file,
                layout_file,
                manifest_desc_file,
            ]),
        ),
    ]

_oci_image = rule(
    implementation = _impl,
    attrs = {
        "base": attr.label(
            mandatory = True,
            providers = [OCIDescriptor, OCILayout],
        ),
        "entrypoint": attr.string_list(),
        "os": attr.string(),
        "arch": attr.string(),
        "env": attr.string_list(),
        "layers": attr.label_list(
            providers = [OCIDescriptor],
        ),
        "annotations": attr.string_dict(),
        "labels": attr.string_dict(),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//go/cmd/ocitool",
            executable = True,
        ),
    },
)
