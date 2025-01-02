""" download utilities """

load("@bazel_skylib//lib:versions.bzl", "versions")

MEDIA_TYPE_DOCKER_INDEX = "application/vnd.docker.distribution.manifest.list.v2+json"
MEDIA_TYPE_DOCKER_MANIFEST = "application/vnd.docker.distribution.manifest.v2+json"
MEDIA_TYPE_OCI_INDEX = "application/vnd.oci.image.index.v1+json"
MEDIA_TYPE_OCI_MANIFEST = "application/vnd.oci.image.manifest.v1+json"

_RESOURCE_BLOB = "blobs"
_RESOURCE_INDEX_OR_MANIFEST = "manifests"

def download_blob(
        rctx,
        *,
        authn,  # authn object
        digest,  # str
        non_blocking = None):  # list[PendingDownload] | None
    # -> struct(path: str, sha256: str) | None
    """Download a blob by digest and write it to the blobs/sha256 directory

    Args:
        rctx: The repository context
        authn: An authn object
        digest: The digest of the blob to download
        non_blocking:
            - If you want the download to block, set this to None (the default)
            - If you want the download to be non-blocking, pass a list here.
              This is an outparam. The function will append a PendingDownload
              object to this list. Later, you can call .wait() on that object
              to block until the download is complete
    """
    sha256 = digest[len("sha256:"):]
    return _download(
        rctx,
        authn = authn,
        digest = digest,
        outpath = rctx.path("blobs/sha256/{}".format(sha256)),
        resource = _RESOURCE_BLOB,
        non_blocking = non_blocking,
    )

def download_index_or_manifest(
        rctx,
        *,
        authn,
        digest,  # str
        outpath,  # str
        non_blocking = None):  # list[PendingDownload] | None
    # -> struct(path: str, sha256: str) | None
    """Download an index or manifest by digest

    Args:
        rctx: The repository context
        authn: An authn object
        digest: The digest of the blob to download
        outpath: The path to write the downloaded index or manifest to
        non_blocking:
            - If you want the download to block, set this to None (the default)
            - If you want the download to be non-blocking, pass a list here.
              This is an outparam. The function will append a PendingDownload
              object to this list. Later, you can call .wait() on that object
              to block until the download is complete
    """
    return _download(
        rctx,
        authn = authn,
        digest = digest,
        outpath = rctx.path(outpath),
        resource = _RESOURCE_INDEX_OR_MANIFEST,
        non_blocking = non_blocking,
    )

def _download(
        rctx,
        *,
        authn,
        digest,  # str
        outpath,  # str
        resource,  # str
        non_blocking = None):  # list[result] | None
    # -> struct(path: str, sha256: str) | None

    non_blocking_type = type(non_blocking)
    if non_blocking_type == type([]):
        block = False
    elif non_blocking_type == type(None):
        block = True
    else:
        fail("Wrong type for non_blocking parameter. Got {non_blocking_type}. Expected a list or None")
    if not digest.startswith("sha256:"):
        fail("Invalid value for digest parameter. Must start with 'sha256:'")

    auth = authn.get_token(rctx.attr.registry, rctx.attr.repository)

    sha256 = digest[len("sha256:"):]

    url = "{scheme}://{registry}/v2/{repository}/{resource}/{digest}".format(
        scheme = rctx.attr.scheme,
        registry = rctx.attr.registry,
        repository = rctx.attr.repository,
        resource = resource,
        digest = digest,
    )

    is_gt_bazel_7_1 = versions.is_at_least("7.1.0", versions.get())

    # On Bazel 7.1.0 and later, use non-blocking download (if requested) and forward headers
    extra_args = {}
    if is_gt_bazel_7_1:
        extra_args["block"] = block
        extra_args["headers"] = {
            "Accept": ",".join([
                MEDIA_TYPE_DOCKER_INDEX,
                MEDIA_TYPE_DOCKER_MANIFEST,
                MEDIA_TYPE_OCI_INDEX,
                MEDIA_TYPE_OCI_MANIFEST,
            ]),
            "Docker-Distribution-API-Version": "registry/2.0",
        }

    res = rctx.download(
        auth = {url: auth},
        output = outpath,
        sha256 = sha256,
        url = url,
        **extra_args
    )

    if is_gt_bazel_7_1 and not block:
        non_blocking.append(res)
        return None

    if not res.success:
        fail(
            "Failed to download OCI object with digest {}\nStdout: {}\nStderr: {}".format(
                digest,
                res.stdout.strip(),
                res.stderr.strip(),
            ),
        )

    return struct(
        path = outpath,
        sha256 = sha256,
    )
