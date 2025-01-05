""" oci_pull """

load(
    "//oci/private:common.bzl",
    "MEDIA_TYPE_DOCKER_INDEX",
    "MEDIA_TYPE_DOCKER_MANIFEST",
    "MEDIA_TYPE_OCI_INDEX",
    "MEDIA_TYPE_OCI_MANIFEST",
)
load(":authn.bzl", _authn = "authn")
load(
    ":download.bzl",
    "download_blob",
    "download_index_or_manifest",
)

_SUPPORTED_PLATFORMS = [
    struct(arch = "amd64", os = "linux", variant = None),
    struct(arch = "arm64", os = "linux", variant = "v8"),
    struct(arch = "arm64", os = "linux", variant = None),
]

def _impl(rctx):
    non_blocking = []  # list of 'PendingDownload's

    authn = _authn.new(rctx, config_path = None)  # authn object

    # Download the index or manifest. At this point, we do not know if the
    # user provided an index or a amanifest. We will determine which it is by
    # inspecting the downloaded file below
    index_or_manifest = download_index_or_manifest(
        rctx,
        authn = authn,
        digest = rctx.attr.digest,
        outpath = "temp.json",
    )

    index_or_manifest_bytes = rctx.read(index_or_manifest.path)
    index_or_manifest_json = json.decode(index_or_manifest_bytes)
    rctx.delete(index_or_manifest.path)

    schema_version = index_or_manifest_json["schemaVersion"]
    if schema_version != 2:
        fail("""
The registry sent a manifest with schemaVersion != 2.
This commonly occurs when fetching from a registry that needs the Docker-Distribution-API-Version header to be set.
See: https://github.com/bazel-contrib/rules_oci/blob/843eb01b152b884fe731a3fb4431b738ad00ea60/docs/pull.md#authentication-using-credential-helpers
        """.strip())

    media_type = index_or_manifest_json["mediaType"]
    if media_type in [
        MEDIA_TYPE_DOCKER_INDEX,
        MEDIA_TYPE_OCI_INDEX,
    ]:
        _handle_index(
            rctx,
            authn = authn,
            index_json = index_or_manifest_json,
            non_blocking = non_blocking,
        )
    elif media_type in [
        MEDIA_TYPE_DOCKER_MANIFEST,
        MEDIA_TYPE_OCI_MANIFEST,
    ]:
        _handle_manifest(
            rctx,
            authn = authn,
            manifest_bytes = index_or_manifest_bytes,
            manifest_json = index_or_manifest_json,
            manifest_sha256 = index_or_manifest.sha256,
            non_blocking = non_blocking,
        )
    else:
        fail(
            """
oci_pull can only be used to pull an image index or an image manifest.
Got media type: {media_type}.
Expected one of: {expected}.
        """.strip().foramt(
                media_type = media_type,
                expected = [
                    MEDIA_TYPE_DOCKER_INDEX,
                    MEDIA_TYPE_DOCKER_MANIFEST,
                    MEDIA_TYPE_OCI_INDEX,
                    MEDIA_TYPE_OCI_MANIFEST,
                ],
            ),
        )

    rctx.file(
        rctx.path("oci-layout"),
        content = json.encode({"imageLayoutVersion": "1.0.0"}),
        executable = False,
    )

    # Wait for all downloads to complete
    for result in non_blocking:
        result.wait()

    # Get the filenames of all the files in the blobs/sha256 directory
    res = rctx.execute(
        ["find", ".", "-maxdepth", "1", "-type", "f"],
        working_directory = "blobs/sha256",
    )
    blobs = [
        s.removeprefix("./")
        for s in res.stdout.strip().split("\n")
    ]

    # Create all the BUILD.bazel files

    rctx.file(
        rctx.path("BUILD.bazel"),
        content = """
exports_files(
    ["index.json", "oci-layout"],
    visibility = ["//:__subpackages__"],
)
""".strip(),
        executable = False,
    )

    rctx.file(
        rctx.path("image/BUILD.bazel"),
        content = """
load(
    "@com_github_datadog_rules_oci//oci/private:oci_pulled_image.bzl",
    "oci_pulled_image",
)

oci_pulled_image(
    name = "image",
    index = "//:index.json",
    blobs = [
{blobs}
    ],
    visibility = ["//visibility:public"],
)
""".strip().format(
            blobs = ",\n".join([
                '        "//blobs/sha256:{}\"'.format(blob)
                for blob in blobs
            ]),
        ),
        executable = False,
    )

    rctx.file(
        rctx.path("blobs/sha256/BUILD.bazel"),
        content = """
exports_files(
    glob(["**/*"]),
    visibility = ["//:__subpackages__"],
)
""".strip(),
        executable = False,
    )

def _handle_index(
        rctx,
        *,
        authn,  # authn object
        index_json,  # dict[str, any]
        non_blocking):  # list[PendingDownload]
    # -> None

    # Filter out unsupported platforms
    original_manifest_jsons = index_json["manifests"][:]
    index_json["manifests"] = []
    for manifest_json in original_manifest_jsons:
        platform = struct(
            arch = manifest_json["platform"]["architecture"],
            os = manifest_json["platform"]["os"],
            variant = manifest_json["platform"].get("variant", None),
        )
        if platform not in _SUPPORTED_PLATFORMS:
            continue
        index_json["manifests"].append(manifest_json)

    index_bytes = json.encode(index_json)

    # Write the index to index.json
    index_path = "index.json"
    rctx.file(index_path, content = index_bytes, executable = False)
    index_sha256 = _sha256sum(rctx, path = index_path)

    # Write the index to the blobs/sha256 directory
    rctx.file(
        "blobs/sha256/{}".format(index_sha256),
        content = index_bytes,
        executable = False,
    )

    # For each manifest
    for item in index_json["manifests"]:
        # Download the manifest
        manifest_digest = item["digest"]
        manifest_sha256 = manifest_digest[len("sha256:"):]
        manifest = download_index_or_manifest(
            rctx,
            authn = authn,
            digest = manifest_digest,
            outpath = "blobs/sha256/{}".format(manifest_sha256),
        )
        manifest_bytes = rctx.read(manifest.path)
        manifest_json = json.decode(manifest_bytes)

        # Download the config
        config_digest = manifest_json["config"]["digest"]
        download_blob(
            rctx,
            authn = authn,
            digest = config_digest,
            non_blocking = non_blocking,
        )

        # Download the layers
        for item in manifest_json["layers"]:
            layer_digest = item["digest"]
            download_blob(
                rctx,
                authn = authn,
                digest = layer_digest,
                non_blocking = non_blocking,
            )

def _handle_manifest(
        rctx,
        *,
        authn,  # authn object
        manifest_bytes,  # bytes
        manifest_json,  # dict[str, any]
        manifest_sha256,  # str
        non_blocking):  # list[PendingDownload]
    # -> None

    # Write the manifest to the blobs directory
    rctx.file(
        "blobs/sha256/{}".format(manifest_sha256),
        content = manifest_bytes,
        executable = False,
    )

    # Download the config
    config_digest = manifest_json["config"]["digest"]
    config = download_blob(rctx, authn = authn, digest = config_digest)
    config_bytes = rctx.read(config.path)
    config_json = json.decode(config_bytes)

    # Read the config for information about the arch/os/variant
    platform = struct(
        arch = config_json["architecture"],
        os = config_json["os"],
        variant = config_json.get("variant", None),
    )

    # Error on unsupported platforms
    if platform not in _SUPPORTED_PLATFORMS:
        fail(
            """
Unsupported platform. Got {}. Expected one of: {supported_platforms}
""".strip().format(
                platform = platform,
                supported_platforms = _SUPPORTED_PLATFORMS,
            ),
        )

    # Create an index and write it to index.json
    index_json = {
        "manifests": [{
            "digest": "sha256:{}".format(manifest_sha256),
            "mediaType": MEDIA_TYPE_OCI_MANIFEST,
            "platform": {
                "architecture": platform.arch,
                "os": platform.os,
            },
            "size": len(manifest_bytes),
        }],
        "mediaType": MEDIA_TYPE_OCI_INDEX,
        "schemaVersion": 2,
    }
    if platform.variant != None:
        index_json["manifests"][0]["platform"]["variant"] = platform.variant

    index_path = "index.json"
    index_bytes = json.encode(index_json)
    rctx.file(index_path, content = index_bytes, executable = False)
    index_sha256 = _sha256sum(rctx, path = index_path)

    # Write the index to the blobs/sha256 directory
    rctx.file(
        "blobs/sha256/{}".format(index_sha256),
        content = index_bytes,
        executable = False,
    )

    # Download the layers
    for item in manifest_json["layers"]:
        layer_digest = item["digest"]
        download_blob(
            rctx,
            authn = authn,
            digest = layer_digest,
            non_blocking = non_blocking,
        )

def _sha256sum(
        rctx,
        *,
        path):  # str
    # -> str
    temp_exe = "sha256-temp.sh"
    rctx.file(
        temp_exe,
        executable = True,
        content = """
#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Wrong number of arguments" >&2
  echo "Usage: $0 <path>" >&2
  exit 1
fi

path="$1"

os="$(uname -s)"
case "$os" in
  Linux) sha256sum "$path" | cut -d ' ' -f 1 ;;
  Darwin) shasum -a 256 "$path"  | cut -d ' ' -f 1 ;;
  *) echo "Unsupported OS. Got $os. Expected of one Darwin or Linux" >&2 ; exit 1 ;;
esac
""".strip().format(
            path = path,
        ),
    )
    res = rctx.execute(["./" + temp_exe, path])
    if res.return_code != 0:
        fail(
            """
Failed to calculate the sha256 of index.json
Exit code: {exit_code}
Stdout: {stdout}
Stderr: {stderr}
""".strip().format(
                exit_code = res.return_code,
                stdout = res.stdout.strip(),
                stderr = res.stderr.strip(),
            ),
        )
    rctx.delete(temp_exe)
    sha256 = res.stdout.strip().split(" ")[0]
    return sha256

oci_pull = repository_rule(
    implementation = _impl,
    attrs = {
        "debug": attr.bool(
            # TODO(brian.myers): Remove this once consumers no longer use it
            doc = "Deprecated. Does nothing",
        ),
        "digest": attr.string(
            doc = "The digest or tag of the manifest file",
            mandatory = True,
        ),
        "registry": attr.string(
            doc = "Remote registry host to pull from, e.g. `gcr.io` or `index.docker.io`",
            mandatory = True,
        ),
        "repository": attr.string(
            doc = "Image path beneath the registry, e.g. `distroless/static`",
            mandatory = True,
        ),
        "scheme": attr.string(
            doc = "scheme portion of the URL for fetching from the registry",
            values = ["http", "https"],
            default = "https",
        ),
        "shallow": attr.bool(
            # TODO(brian.myers): Remove this once consumers no longer use it
            doc = "Deprecated. Does nothing",
        ),
    },
    environ = _authn.ENVIRON,
)
