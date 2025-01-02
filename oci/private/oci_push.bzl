""" oci_push """

load("@aspect_bazel_lib//lib:stamping.bzl", "STAMP_ATTRS", "maybe_stamp")
load("//oci:providers.bzl", "OCIDescriptor", "OCILayout", "OCIReferenceInfo")
load(":debug_flag.bzl", "DebugInfo")

def _impl(ctx):
    layout = ctx.attr.manifest[OCILayout]

    ref = "{registry}/{repository}".format(
        registry = ctx.attr.registry,
        repository = ctx.attr.repository,
    )

    tag_file = ctx.actions.declare_file("{name}.tag".format(name = ctx.label.name))
    ctx.actions.write(
        output = tag_file,
        content = ctx.expand_make_variables("tag", ctx.attr.tag, {}),
    )

    stamp = maybe_stamp(ctx)
    if stamp:
        unstamped_tag_file = tag_file
        tag_file = ctx.actions.declare_file("{name}.stamped.tag".format(name = ctx.label.name))

        args = ctx.actions.args()
        args.add_all([
            unstamped_tag_file,
            tag_file,
            stamp.stable_status_file,
            stamp.volatile_status_file,
        ])

        ctx.actions.run_shell(
            inputs = [
                unstamped_tag_file,
                stamp.stable_status_file,
                stamp.volatile_status_file,
            ],
            outputs = [
                tag_file,
            ],
            arguments = [args],
            command = """#!/usr/bin/env bash
scratch=$(cat $1)
shift

out=$1
shift

for file in $@
do
    while read -r key value
    do
        # Replace the keys with their corresponding values in the scratch output
        scratch=${scratch//\\{$key\\}/$value}
    done <$file
done

>$out echo -n "$scratch"
""",
        )

    digest_file = ctx.actions.declare_file("{name}.digest".format(name = ctx.label.name))
    ctx.actions.run(
        executable = ctx.executable._ocitool,
        arguments = [
            "digest",
            "--desc={desc}".format(desc = ctx.attr.manifest[OCIDescriptor].descriptor_file.path),
            "--out={out}".format(out = digest_file.path),
        ],
        inputs = [
            ctx.attr.manifest[OCIDescriptor].descriptor_file,
        ],
        outputs = [
            digest_file,
        ],
    )

    headers = ""
    for k, v in ctx.attr.headers.items():
        headers = headers + " --headers={}={}".format(k, v)

    xheaders = ""
    for k, v in ctx.attr.x_meta_headers.items():
        xheaders = xheaders + " --x_meta_headers={}={}".format(k, v)

    ctx.actions.write(
        content = """#!/usr/bin/env bash
        set -euo pipefail
        {tool}  \\
        --layout {layout} \\
        --debug={debug} \\
        push \\
        --layout-relative {root} \\
        --desc {desc} \\
        --target-ref {ref} \\
        --parent-tag \"$(cat {tag})\" \\
        {headers} \\
        {xheaders} \\

        export OCI_REFERENCE={ref}@$(cat {digest})
        """.format(
            root = ctx.bin_dir.path,
            tool = ctx.executable._ocitool.short_path,
            layout = layout.blob_index.short_path,
            desc = ctx.attr.manifest[OCIDescriptor].descriptor_file.short_path,
            ref = ref,
            debug = str(ctx.attr._debug[DebugInfo].debug),
            headers = headers,
            xheaders = xheaders,
            digest = digest_file.short_path,
            tag = tag_file.short_path,
        ),
        output = ctx.outputs.executable,
        is_executable = True,
    )

    return [
        DefaultInfo(
            runfiles = ctx.runfiles(
                files = layout.files.to_list() +
                        [
                            ctx.executable._ocitool,
                            ctx.attr.manifest[OCIDescriptor].descriptor_file,
                            layout.blob_index,
                            digest_file,
                            tag_file,
                        ],
            ),
        ),
        OCIReferenceInfo(
            registry = ctx.attr.registry,
            repository = ctx.attr.repository,
            digest = digest_file,
            tag_file = tag_file,
        ),
    ]

oci_push = rule(
    doc = """
        Pushes a manifest or a list of manifests to an OCI registry.
    """,
    implementation = _impl,
    executable = True,
    attrs = dict({
        "manifest": attr.label(
            doc = """
                A manifest to push to a registry. If an OCILayout index, then
                push all artifacts with a 'org.opencontainers.image.ref.name'
                annotation.
            """,
            providers = [OCILayout],
        ),
        "registry": attr.string(
            doc = """
                A registry host to push to, if not present consult the toolchain.
            """,
        ),
        "repository": attr.string(
            doc = """
                A repository to push to, if not present consult the toolchain.
            """,
        ),
        "tag": attr.string(
            doc = """
                (optional) A tag to include in the target reference. This will not be included on child images.

                Subject to [$(location)](https://bazel.build/reference/be/make-variables#predefined_label_variables) and ["Make variable"](https://bazel.build/reference/be/make-variabmes) substitution.

                **Stamping**

                You can use values produced by the workspace status command in your tag. To do this write a script that prints key-value pairs separated by spaces, e.g.

                ```sh
                #!/usr/bin/env bash
                echo "STABLE_KEY1 VALUE1"
                echo "STABLE_KEY2 VALUE2"
                ```

                You can reference these keys in `tag` using curly braces,

                ```python
                oci_push(
                    name = "push",
                    tag = "v1.0-{STABLE_KEY1}",
                )
                ```
            """,
        ),
        "headers": attr.string_dict(
            doc = """
                (optional) A list of key/values to to be sent to the registry as headers.
            """,
        ),
        "x_meta_headers": attr.string_dict(
            doc = """
                (optional) A list of key/values to to be sent to the registry as headers with an X-Meta- prefix.
            """,
        ),
        "_debug": attr.label(
            default = "//oci:debug",
            providers = [DebugInfo],
        ),
        "_ocitool": attr.label(
            allow_single_file = True,
            cfg = "exec",
            default = "//go/cmd/ocitool",
            executable = True,
        ),
    }, **STAMP_ATTRS),
    provides = [
        OCIReferenceInfo,
    ],
)
