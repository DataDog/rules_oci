gazelle:
	bzl run //release:gazelle

release-internal:
	bzl build //release:release
	bzl run //cmd/ocitool -- push-blob --ref "registry.ddbuild.io/public/rules-oci:latest" --file $(shell bzl info bazel-bin)/release/release.tar.gz
