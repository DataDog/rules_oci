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
        "ocitool",
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
        "ocitool",
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
    attrs = {
        "registry": attr.string(
            default = "registry.ddbuild.io",
        ),
        "repository": attr.string(
            mandatory = True,
        ),
        # XXX We're specifically *NOT* supporting pulling by tags as it's
        # difficult for users to control when the tag resolution is done.
        "digest": attr.string(
            mandatory = True,
        ),
        "shallow": attr.bool(
            default = False,
        ),
    },
    environ = [
        OCI_CACHE_DIR_ENV,
    ],
)
