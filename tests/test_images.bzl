load("@com_github_datadog_rules_oci//oci:defs.bzl", "oci_pull")

def pull_test_images():
    oci_pull(
        name = "ubuntu_focal",
        registry = "ghcr.io",
        repository = "datadog/rules_oci/ubuntu",
        # Latest at "focal" tag
        digest = "sha256:9d6a8699fb5c9c39cf08a0871bd6219f0400981c570894cd8cbea30d3424a31f",
    )

    oci_pull(
        name = "tekton-45",
        registry = "gcr.io",
        repository = "tekton-releases/github.com/tektoncd/pipeline/cmd/controller",
        # Latest at "v0.45.0" tag
        digest = "sha256:8a302dab54484bbb83d46ff9455b077ea51c1c189641dcda12575f8301bfb257",
        shallow = False,
    )

    oci_pull(
        name = "old-base-for-rebase",
        registry = "cgr.dev",
        repository = "chainguard/static",
        digest = "sha256:d9dd790fb308621ac4a5d648a852fbc455cda12f487eb30fb775a479c4f90703",
    )

    oci_pull(
        name = "new-base-for-rebase",
        registry = "cgr.dev",
        repository = "chainguard/static",
        digest = "sha256:76bde0b3719bbb65c1b39cd6c0f75fbbe0e24c115a40040ac50361cd8774d913",
    )
