load("@com_github_datadog_rules_oci//oci:defs.bzl", "oci_pull")

def pull_test_images():
    oci_pull(
        name = "ubuntu_focal",
        registry = "ghcr.io",
        repository = "datadog/rules_oci/ubuntu",
        # Latest at "focal" tag
        digest = "sha256:9d6a8699fb5c9c39cf08a0871bd6219f0400981c570894cd8cbea30d3424a31f",
    )

