# A directory to store cached OCI artifacts
# TODO(griffin) currently not used, but going to start depending on this for
# integration into the bzl wrapper.
OCI_CACHE_DIR_ENV = "OCI_CACHE_DIR"

# XXX(griffin): quick hack to get Bazel to spit out debug info for oci_pull
DEBUG = True

def failout(msg, cmd_result):
    fail(
        "{msg}\n stdout: {stdout} \n stderr: {stderr}"
            .format(msg = msg, stdout = cmd_result.stdout, stderr = cmd_result.stderr),
    )

def pull(rctx, layout_root, repository, digest, registry = "", shallow = False):
    cmd = [
        rctx.path(_repo_toolchain(rctx, "ocitool")),
        "--layout={layout_root}".format(layout_root = layout_root),
        "pull",
        "--shallow={shallow}".format(shallow = shallow),
        "{registry}/{repository}@{digest}".format(
            registry = registry,
            repository = repository,
            digest = digest,
        ),
    ]

    res = rctx.execute(cmd, quiet = not DEBUG)
    if res.return_code > 0:
        failout("failed to pull manifest", res)

def generate_build_files(rctx, layout_root, digest=""):
    cmd = [
        rctx.path(_repo_toolchain(rctx, "ocitool")),
        "--debug",
        "--layout={layout_root}".format(layout_root = layout_root),
        "generate-build-files",
        "--image-digest={digest}".format(digest = digest),
    ]

    res = rctx.execute(cmd, quiet = not DEBUG)
    if res.return_code > 0:
        failout("failed to pull manifest", res)

def _oci_pull_impl(rctx):
    pull(
        rctx,
        rctx.path("."),
        repository = rctx.attr.repository,
        digest = rctx.attr.digest,
        registry = rctx.attr.registry,
        shallow = rctx.attr.shallow,
    )

    generate_build_files(
        rctx,
        rctx.path("."),
        digest = rctx.attr.digest,
    )

oci_pull = repository_rule(
    implementation = _oci_pull_impl,
    doc = """
    """,
    attrs = {
        "registry": attr.string(
            mandatory = True,
        doc = """
        """,

        ),
        "repository": attr.string(
            mandatory = True,
            doc = """
            """,
        ),
        # XXX We're specifically *NOT* supporting pulling by tags as it's
        # difficult for users to control when the tag resolution is done.
        "digest": attr.string(
            mandatory = True,
            doc = """
            """,
        ),
        "shallow": attr.bool(
            default = True,
            doc = """
            """,
        ),
        "_ocitool_darwin_amd64": attr.label(
            default = "@com_github_datadog_rules_oci//bin:ocitool-darwin-amd64",
        ),
        "_ocitool_darwin_arm64": attr.label(
            default = "@com_github_datadog_rules_oci//bin:ocitool-darwin-arm64",
        ),
        "_ocitool_linux_amd64": attr.label(
            default = "@com_github_datadog_rules_oci//bin:ocitool-linux-amd64",
        ),
        "_ocitool_linux_arm64": attr.label(
            default = "@com_github_datadog_rules_oci//bin:ocitool-linux-arm64",
        ),
    },
    environ = [
        OCI_CACHE_DIR_ENV,
    ],
)

def _repo_toolchain(rctx, tool_name):
    goos = ""
    goarch = ""

    if rctx.os.name.lower().startswith("mac os"):
        goos = "darwin"
    elif rctx.os.name.lower().startswith("linux"):
        goos = "linux"
    else:
        fail("unknown os: {}".format(rctx.os.name))

    # TODO update to use rctx.os.arch when released
    arch = rctx.execute(["uname", "-m"]).stdout.strip()
    if arch.lower().find("x86") != -1:
        goarch = "amd64"
    elif arch.lower().find("arm64") != -1 or arch.lower().find("aarch64") != -1:
        goarch = "arm64"
    else:
        fail("unknown arch: {}".format(rctx.os.arch))

    return getattr(rctx.attr, "_{tool}_{os}_{arch}".format(tool=tool_name, os=goos, arch=goarch))
