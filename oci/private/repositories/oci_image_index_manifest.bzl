""" oci_image_index_manifest """

load(
    "@com_github_datadog_rules_oci//oci:providers.bzl",
    "OCIDescriptor",
    "OCIImageIndexManifest",
    "OCILayout",
)

def _impl(ctx):
    return [OCIImageIndexManifest(
        manifests = [m[OCIDescriptor] for m in ctx.attr.manifests],
    ), ctx.attr.layout[OCILayout], ctx.attr.descriptor[OCIDescriptor]]

oci_image_index_manifest = rule(
    implementation = _impl,
    attrs = {
        "descriptor": attr.label(
            mandatory = True,
            providers = [OCIDescriptor],
        ),
        "manifests": attr.label_list(
            mandatory = False,
            providers = [OCIDescriptor],
        ),
        "annotations": attr.string_dict(),
        "layout": attr.label(),
    },
    provides = [OCIImageIndexManifest],
)
