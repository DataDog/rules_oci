# Releasing `rules_oci`

```
# Build the release package tar
bzl build //release:release

# Push the package to registry
bzl run //cmd/ocitool -- push-blob --ref "registry.ddbuild.io/public/rules-oci:release" --file $(pwd)/bazel-bin/release/release.tar.gz
```
