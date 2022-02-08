load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCILayoutInfo", "OCIDescriptorInfo", "OCIImageIndexManifestInfo", "OCIImageManifestInfo")

def _oci_image_manifest_impl(ctx):
    return [OCIImageManifestInfo(
        config = ctx.attr.config[OCIDescriptorInfo],
        layers = [l[OCIDescriptorInfo] for l in ctx.attr.layers],
        annotations = ctx.attr.annotations,
    ), ctx.attr.layout[OCILayoutInfo], ctx.attr.descriptor[OCIDescriptorInfo]]

oci_image_manifest = rule(
    implementation = _oci_image_manifest_impl,
    provides = [OCIImageManifestInfo],
    attrs = {
        "descriptor": attr.label(
            mandatory = True,
            providers = [OCIDescriptorInfo],
        ),
        "config": attr.label(
            mandatory = True,
            providers = [OCIDescriptorInfo],
        ),
        "layers": attr.label_list(
            mandatory = False,
            providers = [OCIDescriptorInfo],
        ),
        "annotations": attr.string_dict(),
        "layout": attr.label(),
    },
)

def _oci_image_index_manifest_impl(ctx):
    return [OCIImageIndexManifestInfo(
        manifests = [m[OCIDescriptorInfo] for m in ctx.attr.manifests],
    ), ctx.attr.layout[OCILayoutInfo], ctx.attr.descriptor[OCIDescriptorInfo]]

oci_image_index_manifest = rule(
    implementation = _oci_image_index_manifest_impl,
    attrs = {
        "descriptor": attr.label(
            mandatory = True,
            providers = [OCIDescriptorInfo],
        ),
        "manifests": attr.label_list(
            mandatory = False,
            providers = [OCIDescriptorInfo],
        ),
        "annotations": attr.string_dict(),
        "layout": attr.label(),
    },
    provides = [OCIImageIndexManifestInfo],
)
