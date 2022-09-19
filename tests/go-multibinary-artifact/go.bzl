load("//oci:blob.bzl", "create_blob")
load("//oci:ctx.bzl", "oci_ctx")
load("//oci:image.bzl", "create_oci_image_manifest", "create_oci_index_manifest")
load("//oci:providers.bzl", "OCIPlatform", "OCILayout", "OCIDescriptor")

def platform_triple_to_oci_platform(pt):
    os, arch = pt.split("-")

    return OCIPlatform(
        os = os,
        arch = arch,
    )

def _go_publish_binary_impl(ctx)
    octx = oci_ctx(ctx)

    layouts = []
    manifests = []
    for pt in ctx.attr.platform_triples:
        blob = create_blob(
            octx,
            file = ctx.split_attr.binary[pt].file,
        )

        manifest = create_oci_image_manifest(
            octx,
            layers = blob,
            platform = platform_triple_to_oci_platform(pt)
        )

        layouts.append(manifest.layout)
        manifests.append(manifest_desc)

    index = create_oci_index_manifest(octx, manifests = manifests)

    return [
        index.layout,
        index.index_desc,
    ]

go_publish_binary = rule(
    implementation = _go_publish_binary_impl,
    provides = [OCIDescriptor, OCILayout],
    attrs = {
        "binary": attr.label(
            mandatory = True,
            config = oci_platform_transition,
        ),
        "platform_triples": attr.string_list(),
        # This attribute is required to use Starlark transitions. It allows
        # allowlisting usage of this rule. For more information, see
        # https://docs.bazel.build/skylark/config.html#user-defined-transitions.
        "_allowlist_function_transition": attr.label(
            default = "@bazel_tools//tools/allowlists/function_transition_allowlist",
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)

def _impl(settings, attr):
    return {
        pt : {"//command_line_option:platforms": "@io_bazel_rules_go//go/toolchain:{pt}".format(platform.replace("-", "_"))}
        for pt in attr.platform_triples
    }

oci_platform_transition = transition(
    implementation = _impl,
    inputs = [],
    outputs = ["//command_line_option:platforms"]
)
