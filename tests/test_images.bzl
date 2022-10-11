load("@com_github_datadog_rules_oci//oci:defs.bzl", "oci_pull")

def pull_test_images():
    oci_pull(
        name = "ubuntu_focal",
        registry = "registry.ddbuild.io",
        repository = "base",
        # Latest at "focal" tag
        digest = "sha256:d5f5235357976f0994dfd614b0836c73bb7644505aa7ae4919440324062ddd9c",
    )
