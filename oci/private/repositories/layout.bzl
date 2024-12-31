""" layout """

load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")

def _oci_layout_index_impl(ctx):
    blobs_map = {}
    all_files = []
    for blob in ctx.attr.blobs:
        desc = blob[OCIDescriptor]
        blobs_map[desc.digest] = desc.file.path
        all_files.append(desc.file)

    obj = {
        # TODO
        #"index": ctx.attr.index[OCIDescriptor].file.path,
        "blobs": blobs_map,
    }

    ctx.actions.write(
        output = ctx.outputs.json,
        content = json.encode(obj),
    )

    return [
        OCILayout(
            blob_index = ctx.outputs.json,
            files = depset(all_files),
        ),
    ]

oci_layout_index = rule(
    implementation = _oci_layout_index_impl,
    attrs = {
        "index": attr.label(
            providers = [OCIDescriptor],
        ),
        "blobs": attr.label_list(
            providers = [OCIDescriptor],
        ),
    },
    outputs = {
        "json": "%{name}.layout.json",
    },
    provides = [OCILayout],
)
