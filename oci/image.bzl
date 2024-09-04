""" image """
load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCIImageLayoutInfo", "OCILayout")

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
                        "--bazel-label={}".format(ctx.label),
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
            descriptor_file = descriptor_file,
            file = ctx.outputs.layer,
        ),
    ]

oci_image_layer = rule(
    implementation = _oci_image_layer_impl,
    doc = "Create a tarball and an OCI descriptor for it",
    attrs = {
        "files": attr.label_list(
            doc = "List of files to include under `directory`",
            allow_files = True,
        ),
        "directory": attr.string(
            doc = "Directory in the tarball to place the `files`",
        ),
        "symlinks": attr.string_dict(
            doc = "Dictionary of symlink -> target entries to place in the tarball",
        ),
        "file_map": attr.label_keyed_string_dict(
            doc = "Dictionary of file -> file location in tarball",
            allow_files = True,
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
    outputs = {
        "layer": "%{name}-layer.tar.gz",
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
        inputs = desc_files + layout_files.to_list(),
        outputs = outputs,
    )

    oci_layouts = [m[OCIImageLayoutInfo].oci_image_layout_dirs for m in ctx.attr.manifests]

    return [
        OCIDescriptor(
            descriptor_file = index_desc_file,
        ),
        OCILayout(
            blob_index = layout_file,
            files = depset(direct = [index_file, layout_file], transitive = [layout_files]),
        ),
        # Pass through any OCIImageLayoutInfo data from the manifests.
        OCIImageLayoutInfo(
            oci_image_layout_dirs = depset(transitive = oci_layouts),
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
        executable = toolchain.sdk.ocitool,
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
        OCIImageLayoutInfo(
            oci_image_layout_dirs = depset(ctx.files.pulled_base if ctx.attr.pulled_base != None else []),
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
    implementation = _oci_image_impl,
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
        "pulled_base": attr.label(
            doc = """A directory that contains the base image in OCI Image Layout format.
            See https://github.com/opencontainers/image-spec/blob/main/image-layout.md for a description
            of the OCI Image Layout format. This is optional, and if present, is passed through as an output of oci_image,
            by the OCIImageLayoutInfo provider.""",
            allow_single_file = True,
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
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)
