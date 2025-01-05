# TODO(brian.myers): Delete this file once CNAB tools stops needing it

def register_compiled_oci_toolchains(**kwargs):
    pass

def _oci_toolchain_impl(ctx):
    return [platform_common.ToolchainInfo()]

_oci_toolchain = rule(
    implementation = _oci_toolchain_impl,
    attrs = {
        "sdk": attr.label(),
    },
    provides = [platform_common.ToolchainInfo],
)

def oci_toolchain(
        name,
        **kwargs):
    oci_toolchain_name = "{name}.oci_toolchain".format(name = name)
    _oci_toolchain(
        name = oci_toolchain_name,
        **kwargs
    )

    native.toolchain(
        name = name,
        toolchain = oci_toolchain_name,
        toolchain_type = "@com_github_datadog_rules_oci//oci:toolchain",
    )
