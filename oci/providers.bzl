""" public providers """

OCIReferenceInfo = provider(
    doc = "Refers to any artifact represented by an OCI-like reference URI",
    fields = {
        "registry": "the URI where the artifact is stored",
        "repository": "a namespace for an artifact",
        "tag": "a organizational reference within a repository",
        "tag_file": "a file containing the organizational reference within a repository",
        "digest": "a file containing the digest of the artifact",
    },
)

# buildifier: disable=name-conventions
OCILayout = provider(
    "OCI Layout",
    fields = {
        "blob_index": "",
        "files": "",
    },
)

# buildifier: disable=name-conventions
OCIDescriptor = provider(
    doc = "",
    fields = {
        "file": "A file object of the content this descriptor describes",
        "descriptor_file": "A file object with the information in this provider",
        "media_type": "The MIME media type of the file",
        "size": "The size in bytes of the file",
        "urls": "Additional URLs where you can find the content of file",
        "digest": "A digest, including the algorithm, of the file",
        "annotations": "String map of aribtrary metadata",
    },
)

# buildifier: disable=name-conventions
OCIImageManifest = provider(
    doc = "",
    fields = {
        "config": "Descriptor that points to a configuration object",
        "layers": "List of descriptors",
        "annotations": "String map of arbitrary metadata",
    },
)

# buildifier: disable=name-conventions
OCIImageIndexManifest = provider(
    doc = "",
    fields = {
        "manifests": "List of descriptors",
        "annotations": "String map of arbitrary metadata",
    },
)

OCISDK = provider(
    "The OCI SDK",
    fields = {
        "ocitool": "",
    },
)
