""" oci_image """

load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":common.bzl", "get_descriptor_file")

def _impl(ctx):
    base_desc = get_descriptor_file(ctx, ctx.attr.base[OCIDescriptor])
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

    layer_descriptor_files = [get_descriptor_file(ctx, f[OCIDescriptor]) for f in ctx.attr.layers]
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

oci_image = rule(
    implementation = _impl,
    doc = """Creates a new image manifest and config by appending the `layers` to an existing image
    manifest and config defined by `base`.  If `base` is an image index, then `os` and `arch` will
    be used to extract the image manifest.""",
    attrs = {
        "base": attr.label(
            doc = """A base image, as defined by oci_pull or oci_image""",
            mandatory = True,
            providers = [
                OCIDescriptor,
                OCILayout,
            ],
        ),
        "entrypoint": attr.string_list(
            doc = """A list of entrypoints for the image; these will be inserted into the generated
            OCI image config""",
        ),
        "os": attr.string(
            doc = "Used to extract a manifest from base if base is an index",
        ),
        "arch": attr.string(
            doc = "Used to extract a manifest from base if base is an index",
        ),
        "env": attr.string_list(
            doc = """Entries are in the format of `VARNAME=VARVALUE`. These values act as defaults and
            are merged with any specified when creating a container.""",
        ),
        "layers": attr.label_list(
            doc = "A list of layers defined by oci_image_layer",
            providers = [
                OCIDescriptor,
            ],
        ),
        "annotations": attr.string_dict(
            doc = """[OCI Annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
            to add to the manifest.""",
        ),
        "labels": attr.string_dict(
            doc = """labels that will be applied to the image configuration, as defined in
            [the OCI config](https://github.com/opencontainers/image-spec/blob/main/config.md#properties).
            These behave the same way as
            [docker LABEL](https://docs.docker.com/engine/reference/builder/#label);
            in particular, labels from the base image are inherited.  An empty value for a label
            will cause that label to be deleted.  For backwards compatibility, if this is not set,
            then the value of annotations will be used instead.""",
        ),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//go/cmd/ocitool",
            executable = True,
        ),
    },
)
