""" oci_pulled_image """

load(
    "@com_github_datadog_rules_oci//oci:providers.bzl",
    "OCIDescriptor",
    "OCILayout",
)
load(
    "@com_github_datadog_rules_oci//oci/private:common.bzl",
    "MEDIA_TYPE_OCI_INDEX",
)
load(
    "@com_github_datadog_rules_oci//oci/private:oci_image_dir.bzl",
    "oci_image_dir",
)

_COREUTILS_TOOLCHAIN = "@aspect_bazel_lib//lib:coreutils_toolchain_type"

def oci_pulled_image(
        *,
        name,
        index,  # label
        blobs,  # list[label]
        **kwargs):
    """oci_pulled_image

    Args:
        name: A unique name for this rule
        index: The OCI index.json file
        blobs: A list of the blob files
        **kwargs: Additional arguments to pass to the rule, e.g. tags or visibility
    """
    _oci_pulled_image(
        name = name,
        index = index,
        blobs = blobs,
        **kwargs
    )

    oci_image_dir(
        image = name,
        **kwargs
    )

def _impl(ctx):
    coreutils = ctx.toolchains[_COREUTILS_TOOLCHAIN].coreutils_info.bin

    # Create the descriptor file for the index.json of the image
    descriptor_file = ctx.actions.declare_file("{}.index.descriptor.json".format(ctx.label.name))
    ctx.actions.run_shell(
        inputs = [ctx.file.index],
        outputs = [descriptor_file],
        tools = [coreutils],
        command = """
#!/usr/bin/env bash
set -euo pipefail

coreutils={coreutils}
inpath={inpath}
media_type={media_type}
outpath={outpath}

sha256=$($coreutils sha256sum $inpath | $coreutils cut -d ' ' -f 1)
size=$($coreutils wc -c $inpath | $coreutils cut -d ' ' -f 1)

cat <<EOF > $outpath
{{
  "digest": "sha256:$sha256",
  "mediaType": "$media_type",
  "size": $size
}}
EOF
""".strip().format(
            coreutils = coreutils.path,
            inpath = ctx.file.index.path,
            media_type = MEDIA_TYPE_OCI_INDEX,
            outpath = descriptor_file.path,
        ),
    )

    # Create the "blob index" file (also called a "layout" file) for the image
    blob_index = ctx.actions.declare_file("{}.index.layout.json".format(ctx.label.name))
    ctx.actions.run_shell(
        inputs = ctx.files.blobs,
        outputs = [blob_index],
        command = """
#!/usr/bin/env bash
set -euo pipefail

blobs=({blobs})
outpath={outpath}

cat <<EOF > $outpath
{{
  "blobs": {{
EOF

first=1 # true=1 and false=0
for blob in ${{blobs[@]}}; do
    sha256=$(basename $blob)
    if [[ $first -eq 1 ]]; then
        first=0
    else
        echo "," >> $outpath
    fi
    echo -n "    \\"sha256:$sha256\\": \\"$blob\\"" >> $outpath
done

cat <<EOF >> $outpath

  }}
}}
EOF
""".strip().format(
            blobs = " ".join([f.path for f in ctx.files.blobs]),
            outpath = blob_index.path,
        ),
    )

    return [
        DefaultInfo(
            files = depset([descriptor_file, blob_index]),
        ),
        OCIDescriptor(
            descriptor_file = descriptor_file,
            file = ctx.file.index,
        ),
        OCILayout(
            blob_index = blob_index,
            files = depset(ctx.files.blobs),
        ),
    ]

_oci_pulled_image = rule(
    implementation = _impl,
    attrs = {
        "blobs": attr.label_list(allow_files = True, mandatory = True),
        "index": attr.label(allow_single_file = True, mandatory = True),
    },
    toolchains = [_COREUTILS_TOOLCHAIN],
)
