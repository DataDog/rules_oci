""" config """

load("@com_github_datadog_rules_oci//oci:image.bzl", "get_descriptor_file")
load("@com_github_datadog_rules_oci//oci:providers.bzl", "OCIDescriptor", "OCILayout")

def generate_config_file_action(ctx, config_file, image, os, arch):
    """ Generates a run action with that extracts an image's config file.

    In order to use this action, the calling rule _must_ register
    `@com_github_datadog_rules_oci//oci:toolchain` and the image
    must provide the `OCIDescriptor` and `OCILayout`  (this should
    not be an issue when using the `oci_image` rule).

    Args:
        ctx: The current rules context
        config_file: The file to write the config to
        image: The image to extract the config from.
        os: The os to extract the config for
        arch: The arch to extract the config for

    Returns:
        The config file named after the rule, os, and arch
    """
    toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"]

    base_desc = get_descriptor_file(ctx, image[OCIDescriptor])
    base_layout = image[OCILayout]

    ctx.actions.run(
        executable = toolchain.sdk.ocitool,
        arguments = [
            "--layout={}".format(base_layout.blob_index.path),
            "config",
            "--base={}".format(base_desc.path),
            "--os={}".format(os),
            "--arch={}".format(arch),
            "--out-config={}".format(config_file.path),
        ],
        mnemonic = "OCIImageConfig",
        inputs = [
            base_desc,
            base_layout.blob_index,
        ] + base_layout.files.to_list(),
        outputs = [
            config_file,
        ],
    )

    return config_file

def _oci_image_config_impl(ctx):
    config_file = ctx.actions.declare_file("{}.config.json".format(ctx.label.name))

    return DefaultInfo(files = depset([
        generate_config_file_action(ctx, config_file, ctx.attr.image, ctx.attr.os, ctx.attr.arch),
    ]))

oci_image_config_rule = struct(
    implementation = _oci_image_config_impl,
    attrs = {
        "image": attr.label(
            mandatory = True,
            providers = [OCIDescriptor, OCILayout],
        ),
        "os": attr.string(
            doc = "Used to extract config from image if image is an index",
        ),
        "arch": attr.string(
            doc = "Used to extract config from image if image is an index",
        ),
    },
    toolchains = ["@com_github_datadog_rules_oci//oci:toolchain"],
)

oci_image_config = rule(
    implementation = oci_image_config_rule.implementation,
    attrs = oci_image_config_rule.attrs,
    toolchains = oci_image_config_rule.toolchains,
)
