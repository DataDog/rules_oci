""" manifests """

load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCIImageIndexManifest", "OCIImageManifest", "OCILayout")

def _oci_image_manifest_impl(ctx):
    return [OCIImageManifest(
        config = ctx.attr.config[OCIDescriptor],
        layers = [layer[OCIDescriptor] for layer in ctx.attr.layers],
        annotations = ctx.attr.annotations,
    ), ctx.attr.layout[OCILayout], ctx.attr.descriptor[OCIDescriptor]]

oci_image_manifest = rule(
    implementation = _oci_image_manifest_impl,
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

def _oci_image_index_manifest_impl(ctx):
    return [OCIImageIndexManifest(
        manifests = [m[OCIDescriptor] for m in ctx.attr.manifests],
    ), ctx.attr.layout[OCILayout], ctx.attr.descriptor[OCIDescriptor]]

oci_image_index_manifest = rule(
    implementation = _oci_image_index_manifest_impl,
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
