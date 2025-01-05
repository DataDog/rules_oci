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
    #!/usr/bin/env bash
    set -euxo pipefail
    flags=$([[ ${CI-0} == 1 ]] && echo "--config=ci" || echo "")
    bazel test ${flags} //...
    bazel run ${flags} //:gazelle -- -mode diff || exit 1
    (
        cd examples
        bazel build ${flags} //...
        bazel run ${flags} //:gazelle -- -mode diff || exit 1
    )

update-docs:
    bazel run //docs:update

update-go-3rd-party:
    bazel run //:go -- mod tidy
    bazel run //:gazelle-update-repos

update-rust-3rd-party:
    CARGO_BAZEL_REPIN=1 bazel sync --only=crate_index
