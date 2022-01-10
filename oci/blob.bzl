load("@com_datadoghq_cnab_tools//rules/oci:providers.bzl", "OCIDescriptor")

def _oci_blob_impl(ctx):
    return [OCIDescriptor(
        file = ctx.file.file,
        media_type = ctx.attr.media_type,
        size = ctx.attr.size,
        urls = ctx.attr.urls,
        digest = ctx.attr.digest,
        annotations = ctx.attr.annotations,
    )]

oci_blob = rule(
    implementation = _oci_blob_impl,
    provides = [OCIDescriptor],
    attrs = {
        "file": attr.label(
            allow_single_file = True,
        ),
        "digest": attr.string(),
        "media_type": attr.string(),
        "size": attr.int(),
        "urls": attr.string_list(),
        "annotations": attr.string_dict(),
    },
)
