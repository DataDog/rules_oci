<!-- Generated with Stardoc: http://skydoc.bazel.build -->

public API

<a id="oci_image"></a>

## oci_image

<pre>
load("@rules_oci//oci:defs.bzl", "oci_image")

oci_image(<a href="#oci_image-name">name</a>, <a href="#oci_image-annotations">annotations</a>, <a href="#oci_image-arch">arch</a>, <a href="#oci_image-base">base</a>, <a href="#oci_image-entrypoint">entrypoint</a>, <a href="#oci_image-env">env</a>, <a href="#oci_image-labels">labels</a>, <a href="#oci_image-layers">layers</a>, <a href="#oci_image-os">os</a>, <a href="#oci_image-stamp">stamp</a>)
</pre>

Creates a new image manifest and config by appending the `layers` to an existing image
manifest and config defined by `base`.  If `base` is an image index, then `os` and `arch` will
be used to extract the image manifest.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_image-annotations"></a>annotations |  [OCI Annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md) to add to the manifest.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="oci_image-arch"></a>arch |  Used to extract a manifest from base if base is an index   | String | optional |  `""`  |
| <a id="oci_image-base"></a>base |  A base image, as defined by oci_pull or oci_image   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="oci_image-entrypoint"></a>entrypoint |  A list of entrypoints for the image; these will be inserted into the generated OCI image config   | List of strings | optional |  `[]`  |
| <a id="oci_image-env"></a>env |  Entries are in the format of `VARNAME=VARVALUE`. These values act as defaults and are merged with any specified when creating a container.   | List of strings | optional |  `[]`  |
| <a id="oci_image-labels"></a>labels |  labels that will be applied to the image configuration, as defined in [the OCI config](https://github.com/opencontainers/image-spec/blob/main/config.md#properties). These behave the same way as [docker LABEL](https://docs.docker.com/engine/reference/builder/#label); in particular, labels from the base image are inherited.  An empty value for a label will cause that label to be deleted.  For backwards compatibility, if this is not set, then the value of annotations will be used instead.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="oci_image-layers"></a>layers |  A list of layers defined by oci_image_layer   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |
| <a id="oci_image-os"></a>os |  Used to extract a manifest from base if base is an index   | String | optional |  `""`  |
| <a id="oci_image-stamp"></a>stamp |  Whether to encode build information into the output. Possible values:<br><br>- `stamp = 1`: Always stamp the build information into the output, even in     [--nostamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) builds.     This setting should be avoided, since it is non-deterministic.     It potentially causes remote cache misses for the target and     any downstream actions that depend on the result. - `stamp = 0`: Never stamp, instead replace build information by constant values.     This gives good build result caching. - `stamp = -1`: Embedding of build information is controlled by the     [--[no]stamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) flag.     Stamped targets are not rebuilt unless their dependencies change.   | Integer | optional |  `-1`  |


<a id="oci_image_config"></a>

## oci_image_config

<pre>
load("@rules_oci//oci:defs.bzl", "oci_image_config")

oci_image_config(<a href="#oci_image_config-name">name</a>, <a href="#oci_image_config-arch">arch</a>, <a href="#oci_image_config-image">image</a>, <a href="#oci_image_config-os">os</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_config-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_image_config-arch"></a>arch |  Used to extract config from image if image is an index   | String | optional |  `""`  |
| <a id="oci_image_config-image"></a>image |  -   | <a href="https://bazel.build/concepts/labels">Label</a> | required |  |
| <a id="oci_image_config-os"></a>os |  Used to extract config from image if image is an index   | String | optional |  `""`  |


<a id="oci_image_index"></a>

## oci_image_index

<pre>
load("@rules_oci//oci:defs.bzl", "oci_image_index")

oci_image_index(<a href="#oci_image_index-name">name</a>, <a href="#oci_image_index-annotations">annotations</a>, <a href="#oci_image_index-manifests">manifests</a>)
</pre>



**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_index-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_image_index-annotations"></a>annotations |  -   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="oci_image_index-manifests"></a>manifests |  -   | <a href="https://bazel.build/concepts/labels">List of labels</a> | optional |  `[]`  |


<a id="oci_image_layout"></a>

## oci_image_layout

<pre>
load("@rules_oci//oci:defs.bzl", "oci_image_layout")

oci_image_layout(<a href="#oci_image_layout-name">name</a>, <a href="#oci_image_layout-manifest">manifest</a>)
</pre>

Writes an OCI Image Index and related blobs to an OCI Image Format
directory. See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
for the specification of the OCI Image Format directory.

All blobs must be provided in the manifest's OCILayout provider, in the
files attribute. If blobs are missing, creation of the OCI Image Layout
will fail.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_layout-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_image_layout-manifest"></a>manifest |  An OCILayout index to be written to the OCI Image Format directory.   | <a href="https://bazel.build/concepts/labels">Label</a> | optional |  `None`  |


<a id="oci_push"></a>

## oci_push

<pre>
load("@rules_oci//oci:defs.bzl", "oci_push")

oci_push(<a href="#oci_push-name">name</a>, <a href="#oci_push-headers">headers</a>, <a href="#oci_push-manifest">manifest</a>, <a href="#oci_push-registry">registry</a>, <a href="#oci_push-repository">repository</a>, <a href="#oci_push-stamp">stamp</a>, <a href="#oci_push-tag">tag</a>, <a href="#oci_push-x_meta_headers">x_meta_headers</a>)
</pre>

Pushes a manifest or a list of manifests to an OCI registry.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_push-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_push-headers"></a>headers |  (optional) A list of key/values to to be sent to the registry as headers.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  `{}`  |
| <a id="oci_push-manifest"></a>manifest |  A manifest to push to a registry. If an OCILayout index, then push all artifacts with a 'org.opencontainers.image.ref.name' annotation.   | <a href="https://bazel.build/concepts/labels">Label</a> | optional |  `None`  |
| <a id="oci_push-registry"></a>registry |  A registry host to push to, if not present consult the toolchain.   | String | optional |  `""`  |
| <a id="oci_push-repository"></a>repository |  A repository to push to, if not present consult the toolchain.   | String | optional |  `""`  |
| <a id="oci_push-stamp"></a>stamp |  Whether to encode build information into the output. Possible values:<br><br>- `stamp = 1`: Always stamp the build information into the output, even in     [--nostamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) builds.     This setting should be avoided, since it is non-deterministic.     It potentially causes remote cache misses for the target and     any downstream actions that depend on the result. - `stamp = 0`: Never stamp, instead replace build information by constant values.     This gives good build result caching. - `stamp = -1`: Embedding of build information is controlled by the     [--[no]stamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) flag.     Stamped targets are not rebuilt unless their dependencies change.   | Integer | optional |  `-1`  |
| <a id="oci_push-tag"></a>tag |  (optional) A tag to include in the target reference. This will not be included on child images.<br><br>Subject to [$(location)](https://bazel.build/reference/be/make-variables#predefined_label_variables) and ["Make variable"](https://bazel.build/reference/be/make-variabmes) substitution.<br><br>**Stamping**<br><br>You can use values produced by the workspace status command in your tag. To do this write a script that prints key-value pairs separated by spaces, e.g.<br><br><pre><code class="language-sh">#!/usr/bin/env bash&#10;echo "STABLE_KEY1 VALUE1"&#10;echo "STABLE_KEY2 VALUE2"</code></pre><br><br>You can reference these keys in `tag` using curly braces,<br><br><pre><code class="language-python">oci_push(&#10;    name = "push",&#10;    tag = "v1.0-{STABLE_KEY1}",&#10;)</code></pre>   | String | optional |  `""`  |
| <a id="oci_push-x_meta_headers"></a>x_meta_headers |  (optional) A list of key/values to to be sent to the registry as headers with an X-Meta- prefix.   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  `{}`  |


<a id="generate_config_file_action"></a>

## generate_config_file_action

<pre>
load("@rules_oci//oci:defs.bzl", "generate_config_file_action")

generate_config_file_action(<a href="#generate_config_file_action-ctx">ctx</a>, <a href="#generate_config_file_action-config_file">config_file</a>, <a href="#generate_config_file_action-image">image</a>, <a href="#generate_config_file_action-os">os</a>, <a href="#generate_config_file_action-arch">arch</a>)
</pre>

Generates a run action with that extracts an image's config file.

In order to use this action, the calling rule _must_ register
`@com_github_datadog_rules_oci//oci:toolchain` and the image
must provide the `OCIDescriptor` and `OCILayout`  (this should
not be an issue when using the `oci_image` rule).


**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="generate_config_file_action-ctx"></a>ctx |  The current rules context   |  none |
| <a id="generate_config_file_action-config_file"></a>config_file |  The file to write the config to   |  none |
| <a id="generate_config_file_action-image"></a>image |  The image to extract the config from.   |  none |
| <a id="generate_config_file_action-os"></a>os |  The os to extract the config for   |  none |
| <a id="generate_config_file_action-arch"></a>arch |  The arch to extract the config for   |  none |

**RETURNS**

The config file named after the rule, os, and arch


<a id="oci_image_layer"></a>

## oci_image_layer

<pre>
load("@rules_oci//oci:defs.bzl", "oci_image_layer")

oci_image_layer(<a href="#oci_image_layer-name">name</a>, <a href="#oci_image_layer-directory">directory</a>, <a href="#oci_image_layer-files">files</a>, <a href="#oci_image_layer-file_map">file_map</a>, <a href="#oci_image_layer-mode_map">mode_map</a>, <a href="#oci_image_layer-owner_map">owner_map</a>, <a href="#oci_image_layer-symlinks">symlinks</a>, <a href="#oci_image_layer-compression_method">compression_method</a>,
                <a href="#oci_image_layer-kwargs">kwargs</a>)
</pre>

Creates a tarball and an OCI descriptor for it

**PARAMETERS**


| Name  | Description | Default Value |
| :------------- | :------------- | :------------- |
| <a id="oci_image_layer-name"></a>name |  A unique name for this rule   |  none |
| <a id="oci_image_layer-directory"></a>directory |  Directory in the tarball to place the `files`   |  `None` |
| <a id="oci_image_layer-files"></a>files |  List of files to include under `directory`   |  `None` |
| <a id="oci_image_layer-file_map"></a>file_map |  Dictionary of file -> file location in tarball   |  `None` |
| <a id="oci_image_layer-mode_map"></a>mode_map |  Dictionary of file location in tarball -> mode int (e.g. 0o755)   |  `None` |
| <a id="oci_image_layer-owner_map"></a>owner_map |  Dictionary of file location in tarball -> owner:group string (e.g. '501:501')   |  `None` |
| <a id="oci_image_layer-symlinks"></a>symlinks |  Dictionary of symlink -> target entries to place in the tarball   |  `None` |
| <a id="oci_image_layer-compression_method"></a>compression_method |  A string, currently supports "gzip" and "zstd", defaults to "gzip"   |  `"gzip"` |
| <a id="oci_image_layer-kwargs"></a>kwargs |  Additional arguments to pass to the rule, e.g. tags or visibility   |  none |


<a id="oci_pull"></a>

## oci_pull

<pre>
load("@rules_oci//oci:defs.bzl", "oci_pull")

oci_pull(<a href="#oci_pull-name">name</a>, <a href="#oci_pull-debug">debug</a>, <a href="#oci_pull-digest">digest</a>, <a href="#oci_pull-registry">registry</a>, <a href="#oci_pull-repo_mapping">repo_mapping</a>, <a href="#oci_pull-repository">repository</a>, <a href="#oci_pull-shallow">shallow</a>)
</pre>

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_pull-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/concepts/labels#target-names">Name</a> | required |  |
| <a id="oci_pull-debug"></a>debug |  Enable ocitool debug output   | Boolean | optional |  `False`  |
| <a id="oci_pull-digest"></a>digest |  -   | String | required |  |
| <a id="oci_pull-registry"></a>registry |  -   | String | required |  |
| <a id="oci_pull-repo_mapping"></a>repo_mapping |  In `WORKSPACE` context only: a dictionary from local repository name to global repository name. This allows controls over workspace dependency resolution for dependencies of this repository.<br><br>For example, an entry `"@foo": "@bar"` declares that, for any time this repository depends on `@foo` (such as a dependency on `@foo//some:target`, it should actually resolve that dependency within globally-declared `@bar` (`@bar//some:target`).<br><br>This attribute is _not_ supported in `MODULE.bazel` context (when invoking a repository rule inside a module extension's implementation function).   | <a href="https://bazel.build/rules/lib/dict">Dictionary: String -> String</a> | optional |  |
| <a id="oci_pull-repository"></a>repository |  -   | String | required |  |
| <a id="oci_pull-shallow"></a>shallow |  -   | Boolean | optional |  `True`  |

**ENVIRONMENT VARIABLES**

This repository rule depends on the following environment variables:

* `OCI_CACHE_DIR`


