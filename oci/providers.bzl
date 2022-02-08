OCITOOL_ATTR = attr.label(
    executable = True,
    default = "@com_github_datadog_rules_oci//cmd/ocitool:ocitool",
    cfg = "host",
)

OCIReferenceInfo = provider(
    doc = "Refers to any artifact represented by an OCI-like reference URI",
    fields = {
        "registry": "the URI where the artifact is stored",
        "repository": "a namespace for an artifact",
        "tag": "a organizational reference within a repository",
        "digest": "a file containing the digest of the artifact",
    },
)

OCILayoutInfo = provider(
    fields = {
        "blob_index": "",
        "files": "",
    },
)

OCIDescriptorInfo = provider(
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

OCIImageManifestInfo = provider(
    doc = "",
    fields = {
        "config": "Descriptor that points to a configuration object",
        "layers": "List of descriptors",
        "annotations": "String map of arbitrary metadata",
    },
)

OCIImageIndexManifestInfo = provider(
    doc = "",
    fields = {
        "manifests": "List of desciptors",
        "annotations": "String map of arbitrary metadata",
    },
)

OCIPlatformInfo = provider(
    doc = "Platform describes the platform which the image in the manifest runs on",
    fields = {
        "architecture": "Architecture field specifies the CPU architecture, in the GOARCH format.",
        "os": "OS specifies the operating system, in the GOOS format.",
        "os_version": "OSVersion is an optional field specifying the operating system version",
        "os_features": "OSFeatures is an optional field specifying an array of strings, each listing a required OS feature",
        "variant": "Variant is an optional field specifying a variant of the CPU",
    },
)
