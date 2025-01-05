build:
    bazel build //...

docs-rust:
    #!/usr/bin/env bash
    set -euo pipefail
    root="$(git rev-parse --show-toplevel)"
    bazel build //ocitool:docs
    open "${root}/bazel-bin/ocitool/docs.rustdoc/ocitool/index.html"

format:
    bazel run //:format

gazelle:
    bazel run //:gazelle

release: test
    #!/usr/bin/env bash
    set -euo pipefail
    root="$(git rev-parse --show-toplevel)"
    bazel build //:release
    bazel run //go/cmd/ocitool -- \
        push-blob \
        --file "${root}/bazel-bin/release.tar.gz" \
        --ref "ghcr.io/datadog/rules_oci/rules:latest"

test:
    bazel test //...
    bazel run //:gazelle -- -mode diff || exit 1

update-crates:
    CARGO_BAZEL_REPIN=1 bazel sync --only=crate_index

update-docs:
    bazel run //docs:update

foobar:
    bzl run //examples/simple:image.load
