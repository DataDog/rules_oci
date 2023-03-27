# Releasing `rules_oci`

```
# Build the release package tar
bzl build //release:release

# Push the package to registry
bzl run //go/cmd/ocitool -- push-blob --ref "ghcr.io/datadog/rules_oci/rules:latest" --file $(pwd)/bazel-bin/release/release.tar.gz
```

## Updating Licenses and Headers

```
go install github.com/DataDog/temporalite/internal/licensecheck@latest
go install github.com/DataDog/temporalite/internal/copyright@latest
licensecheck
copyright
```
