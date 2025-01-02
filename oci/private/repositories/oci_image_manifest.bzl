""" oci_image_manifest """

load(
    "@com_github_datadog_rules_oci//oci:providers.bzl",
    "OCIDescriptor",
    "OCIImageManifest",
    "OCILayout",
)

def _impl(ctx):
    return [OCIImageManifest(
        config = ctx.attr.config[OCIDescriptor],
        layers = [layer[OCIDescriptor] for layer in ctx.attr.layers],
        annotations = ctx.attr.annotations,
    ), ctx.attr.layout[OCILayout], ctx.attr.descriptor[OCIDescriptor]]

oci_image_manifest = rule(
    implementation = _impl,
    provides = [OCIImageManifest],
    attrs = {
        "descriptor": attr.label(
            mandatory = True,
            providers = [OCIDescriptor],
        ),
        "config": attr.label(
            mandatory = True,
            providers = [OCIDescriptor],
        ),
        "layers": attr.label_list(
            mandatory = False,
            providers = [OCIDescriptor],
        ),
        "annotations": attr.string_dict(),
        "layout": attr.label(),
    },
)
