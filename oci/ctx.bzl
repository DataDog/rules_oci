def oci_ctx(ctx):
    return struct(
        prefix = ctx.label.name,
        toolchain = ctx.toolchains["@com_github_datadog_rules_oci//oci:toolchain"],
        actions = ctx.actions,
    )
