load(":providers.bzl", "OCIPlatformInfo")

def _oci_platform_impl(ctx):
    return [
        OCIPlatformInfo(
            os = ctx.attr.os,
            architecture = ctx.attr.architecture,
            os_version = ctx.attr.os_version,
            os_features = ctx.attr.os_features,
            variant = ctx.attr.variant,
        ),
    ]

oci_platform = rule(
    implementation = _oci_platform_impl,
    attrs = {
        "os": attr.string(
            doc = """
OS specifies the operating system, in the GOOS format.
            """,
        ),
        "architecture": attr.string(
            doc = """
Architecture field specifies the CPU architecture, in the GOARCH format.
            """,
        ),
        "os_version": attr.string(
            doc = """
OSVersion is an optional field specifying the operating system version.
            """,
        ),
        "os_features": attr.string_list(
            doc = """
OSFeatures is an optional field specifying an array of strings, each listing a required OS feature.
            """,
        ),
        "variant": attr.string(
            doc = """
Variant is an optional field specifying a variant of the CPU.
            """,
        ),
    },
    provides = [
        OCIPlatformInfo,
    ],
)
