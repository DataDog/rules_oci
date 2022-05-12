# Releasing `rules_oci`

```
# Build the release package tar
bzl build //release:release

# Push the package to registry
bzl run //cmd/ocitool -- push-blob --ref "ghcr.io/datadog/rules_oci/rules:latest" --file $(pwd)/bazel-bin/release/release.tar.gz
```
