""" 3rd party go and rust libraries """

load("@com_github_datadog_rules_oci//oci/private/repositories:go_repositories.bzl", "go_repositories")
load("@com_github_datadog_rules_oci_crate_index//:defs.bzl", "crate_repositories")

def rules_oci_third_party_go_and_rust_libraries():
    crate_repositories()
    go_repositories()
