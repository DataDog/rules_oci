""" public providers """

# buildifier: disable=name-conventions
OCIDescriptor = provider(
    doc = "An OCI descriptor. See https://github.com/opencontainers/image-spec/blob/main/descriptor.md",
    fields = {
        "file": "A file object of the content this descriptor describes",
        "descriptor_file": "A file object with the information in this provider",
        #
        "artifact_type": "Optional. The type of an artifact when the descriptor points to an artifact",
        "data": "Optional. An embedded representation of the referenced content",
        "annotations": "Optional. Arbitrary metadata for this descriptor",
        "digest": "Required. The digest of the targeted content",
        "media_type": "Required. The media type of the referenced content",
        "size": "Required. The size, in bytes, of the raw content",
        "urls": "Optional. A list of URIs from which this object MAY be downloaded",
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
