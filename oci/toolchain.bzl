# Follow golang's conventions for naming os and arch
OS_CONSTRAINTS = {
    "darwin": "@platforms//os:osx",
    "linux": "@platforms//os:linux",
}

ARCH_CONSTRAINTS = {
    "amd64": "@platforms//cpu:x86_64",
    "arm64": "@platforms//cpu:arm64",
}

OCISDK = provider(
    fields = {
        "ocitool": "",
    },
)

def _oci_toolchain_impl(ctx):
    return [platform_common.ToolchainInfo(
        sdk = ctx.attr.sdk[OCISDK],
        post_push_hooks = ctx.files.post_push_hooks,
    )]

_oci_toolchain = rule(
    implementation = _oci_toolchain_impl,
    attrs = {
        "sdk": attr.label(
            mandatory = True,
            providers = [OCISDK],
            cfg = "exec",
        ),
        "post_push_hooks": attr.label_list(
            cfg = "exec",
            allow_files = True,
        ),
    },
    provides = [platform_common.ToolchainInfo],
)

def oci_toolchain(
        name,
        exec_compatible_with = [],
        target_compatible_with = [],
        **kwargs):
    oci_toolchain_name = "{name}.oci_toolchain".format(name = name)
    _oci_toolchain(
        name = oci_toolchain_name,
        **kwargs,
    )

    native.toolchain(
        name = name,
        toolchain = oci_toolchain_name,
        exec_compatible_with = exec_compatible_with,
        target_compatible_with = target_compatible_with,
        toolchain_type = "@com_github_datadog_rules_oci//oci:toolchain",
    )

def _oci_sdk_impl(ctx):
    return [
        OCISDK(
            ocitool = ctx.executable.ocitool,
        ),
    ]

oci_sdk = rule(
    implementation = _oci_sdk_impl,
    attrs = {
        "ocitool": attr.label(
            mandatory = True,
            allow_single_file = True,
            executable = True,
            cfg = "host",
        ),
    },
    provides = [OCISDK],
)

def oci_local_toolchain(name, **kwargs):
    sdk_name = "{}.sdk".format(name)
    oci_sdk(
        name = sdk_name,
        ocitool = "@com_github_datadog_rules_oci//cmd/ocitool",
    )

    oci_toolchain(
        name = name,
        sdk = sdk_name,
    )

def create_compiled_oci_toolchains(name, **kwargs):
    for os, os_const in OS_CONSTRAINTS.items():
        for arch, arch_const in ARCH_CONSTRAINTS.items():
            sdk_name = "{name}_sdk_{os}_{arch}".format(name = name, os = os, arch = arch)
            oci_sdk(
                name = sdk_name,
                ocitool = "@com_github_datadog_rules_oci//bin:ocitool-{os}-{arch}".format(os = os, arch = arch),
            )

            toolchain_name = "{name}_toolchain_{os}_{arch}".format(name = name, os = os, arch = arch)
            oci_toolchain(
                name = toolchain_name,
                sdk = sdk_name,
                exec_compatible_with = [
                    os_const,
                    arch_const,
                ],
                **kwargs
            )

def register_compiled_oci_toolchains(name, post_push_hooks=[]):
    registry_post_push_hooks(
        name = "oci_push_hooks",
        post_push_hooks = post_push_hooks,
    )

    for os, os_const in OS_CONSTRAINTS.items():
        for arch, arch_const in ARCH_CONSTRAINTS.items():
            toolchain_name = "{name}_toolchain_{os}_{arch}".format(name = name, os = os, arch = arch)
            native.register_toolchains("@com_github_datadog_rules_oci//bin:{}".format(toolchain_name))


def _registry_post_push_hooks_impl(rctx):
    rctx.file("defs.bzl", content = """
POST_PUSH_HOOKS = {post_push_hooks}
    """.format(
        post_push_hooks = json.encode(rctx.attr.post_push_hooks),
    ))

    rctx.file("BUILD.bazel")

registry_post_push_hooks = repository_rule(
    implementation = _registry_post_push_hooks_impl,
    attrs = {
        "post_push_hooks": attr.string_list(),
    },
)
