""" oci_image_index """

load("//oci:providers.bzl", "OCIDescriptor", "OCILayout")
load(":common.bzl", "get_descriptor_file")

def _impl(ctx):
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
        executable = ctx.executable._ocitool,
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
    implementation = _impl,
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
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//go/cmd/ocitool",
            executable = True,
        ),
    },
)
