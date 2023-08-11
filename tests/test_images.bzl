load("@com_github_datadog_rules_oci//oci:defs.bzl", "oci_pull")

def pull_test_images():
    oci_pull(
        name = "ubuntu_focal",
        registry = "ghcr.io",
        repository = "datadog/rules_oci/ubuntu",
        # Latest at "focal" tag
        digest = "sha256:9d6a8699fb5c9c39cf08a0871bd6219f0400981c570894cd8cbea30d3424a31f",
    )

    # TODO(abayer): Temporarily using a public image I pushed to my own gcr.io repo.
    oci_pull(
        name = "ubuntu_jammy",
        registry = "gcr.io",
        repository = "abayer-jclouds-test1/rules_oci/ubuntu",
        # Latest at "jammy" tag
        digest = "sha256:99f98de8a0a27a7e1b3979238d17422ae3359573bda3beed0906da7e2d42e8c3",
    )
