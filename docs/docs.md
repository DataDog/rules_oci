<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="#oci_image"></a>

## oci_image

<pre>
oci_image(<a href="#oci_image-name">name</a>, <a href="#oci_image-annotations">annotations</a>, <a href="#oci_image-arch">arch</a>, <a href="#oci_image-base">base</a>, <a href="#oci_image-entrypoint">entrypoint</a>, <a href="#oci_image-env">env</a>, <a href="#oci_image-labels">labels</a>, <a href="#oci_image-layers">layers</a>, <a href="#oci_image-os">os</a>)
</pre>

Creates a new image manifest and config by appending the `layers` to an existing image
    manifest and config defined by `base`.  If `base` is an image index, then `os` and `arch` will
    be used to extract the image manifest.

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image-annotations"></a>annotations |  [OCI Annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)             to add to the manifest.   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |
| <a id="oci_image-arch"></a>arch |  Used to extract a manifest from base if base is an index   | String | optional | "" |
| <a id="oci_image-base"></a>base |  A base image, as defined by oci_pull or oci_image   | <a href="https://bazel.build/docs/build-ref.html#labels">Label</a> | required |  |
| <a id="oci_image-entrypoint"></a>entrypoint |  A list of entrypoints for the image; these will be inserted into the generated             OCI image config   | List of strings | optional | [] |
| <a id="oci_image-env"></a>env |  Entries are in the format of <code>VARNAME=VARVALUE</code>. These values act as defaults and              are merged with any specified when creating a container.   | List of strings | optional | [] |
| <a id="oci_image-labels"></a>labels |  labels that will be applied to the image configuration, as defined in             [the OCI config](https://github.com/opencontainers/image-spec/blob/main/config.md#properties).             These behave the same way as             [docker LABEL](https://docs.docker.com/engine/reference/builder/#label);             in particular, labels from the base image are inherited.  An empty value for a label             will cause that label to be deleted.  For backwards compatibility, if this is not set,             then the value of annotations will be used instead.   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |
| <a id="oci_image-layers"></a>layers |  A list of layers defined by oci_image_layer   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="oci_image-os"></a>os |  Used to extract a manifest from base if base is an index   | String | optional | "" |


<a id="#oci_image_index"></a>

## oci_image_index

<pre>
oci_image_index(<a href="#oci_image_index-name">name</a>, <a href="#oci_image_index-annotations">annotations</a>, <a href="#oci_image_index-manifests">manifests</a>)
</pre>


    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_index-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image_index-annotations"></a>annotations |     | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |
| <a id="oci_image_index-manifests"></a>manifests |     | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |


<a id="#oci_image_layer"></a>

## oci_image_layer

<pre>
oci_image_layer(<a href="#oci_image_layer-name">name</a>, <a href="#oci_image_layer-directory">directory</a>, <a href="#oci_image_layer-file_map">file_map</a>, <a href="#oci_image_layer-files">files</a>, <a href="#oci_image_layer-symlinks">symlinks</a>)
</pre>

Create a tarball and an OCI descriptor for it

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_layer-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image_layer-directory"></a>directory |  Directory in the tarball to place the <code>files</code>   | String | optional | "" |
| <a id="oci_image_layer-file_map"></a>file_map |  Dictionary of file -&gt; file location in tarball   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: Label -> String</a> | optional | {} |
| <a id="oci_image_layer-files"></a>files |  List of files to include under <code>directory</code>   | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="oci_image_layer-symlinks"></a>symlinks |  Dictionary of symlink -&gt; target entries to place in the tarball   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |


<a id="#oci_image_layout"></a>

## oci_image_layout

<pre>
oci_image_layout(<a href="#oci_image_layout-name">name</a>, <a href="#oci_image_layout-manifest">manifest</a>, <a href="#oci_image_layout-registry">registry</a>, <a href="#oci_image_layout-repository">repository</a>)
</pre>


        Writes an OCI Image Index and related blobs to an OCI Image Format
        directory. See https://github.com/opencontainers/image-spec/blob/main/image-layout.md
        for the specification of the OCI Image Format directory. Local blobs are
        used where available, and if a referenced blob is not present, it is
        fetched from the provided OCI repository and placed in the output.

        In order for this rule to work correctly in its current state, the
        following flags must be provided to bazel:
        --incompatible_strict_action_env=false
        --spawn_strategy=local

        The incompatible_strict_action_env flag is required because in order to
        access the registry, a credential helper executable (named
        docker-credential-<SOMETHING>) must be available so that ocitool can
        execute it. The incompatible_strict_action_env flag makes the system
        PATH available to bazel rules.

        The spawn_strategy flag must be set to local because currently,
        oci_image_index is only declaring the new JSON files it creates as
        outputs; it's not declaring any manifests or layers from the images as
        outputs. By default, Bazel only permits rules to access specifically
        declared outputs of the rule's direct dependencies. In order for this
        rule to access the transitive set of outputs of all dependencies, we
        must disable bazel's sandboxing by setting spawn_strategy=local.
    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_layout-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image_layout-manifest"></a>manifest |  An OCILayout index to be written to the OCI Image Format directory.   | <a href="https://bazel.build/docs/build-ref.html#labels">Label</a> | optional | None |
| <a id="oci_image_layout-registry"></a>registry |  A registry host that contains images referenced by the OCILayout index,                 if not present consult the toolchain.   | String | optional | "" |
| <a id="oci_image_layout-repository"></a>repository |  A repository that contains images referenced by the OCILayout index,                 if not present consult the toolchain.   | String | optional | "" |


<a id="#oci_pull"></a>

## oci_pull

<pre>
oci_pull(<a href="#oci_pull-name">name</a>, <a href="#oci_pull-digest">digest</a>, <a href="#oci_pull-registry">registry</a>, <a href="#oci_pull-repo_mapping">repo_mapping</a>, <a href="#oci_pull-repository">repository</a>, <a href="#oci_pull-shallow">shallow</a>)
</pre>


    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_pull-name"></a>name |  A unique name for this repository.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_pull-digest"></a>digest |     | String | required |  |
| <a id="oci_pull-registry"></a>registry |     | String | required |  |
| <a id="oci_pull-repo_mapping"></a>repo_mapping |  A dictionary from local repository name to global repository name. This allows controls over workspace dependency resolution for dependencies of this repository.&lt;p&gt;For example, an entry <code>"@foo": "@bar"</code> declares that, for any time this repository depends on <code>@foo</code> (such as a dependency on <code>@foo//some:target</code>, it should actually resolve that dependency within globally-declared <code>@bar</code> (<code>@bar//some:target</code>).   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | required |  |
| <a id="oci_pull-repository"></a>repository |     | String | required |  |
| <a id="oci_pull-shallow"></a>shallow |     | Boolean | optional | True |


<a id="#oci_push"></a>

## oci_push

<pre>
oci_push(<a href="#oci_push-name">name</a>, <a href="#oci_push-headers">headers</a>, <a href="#oci_push-manifest">manifest</a>, <a href="#oci_push-registry">registry</a>, <a href="#oci_push-repository">repository</a>, <a href="#oci_push-stamp">stamp</a>, <a href="#oci_push-tag">tag</a>, <a href="#oci_push-x_meta_headers">x_meta_headers</a>)
</pre>


        Pushes a manifest or a list of manifests to an OCI registry.
    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_push-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_push-headers"></a>headers |  (optional) A list of key/values to to be sent to the registry as headers.   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |
| <a id="oci_push-manifest"></a>manifest |  A manifest to push to a registry. If an OCILayout index, then                 push all artifacts with a 'org.opencontainers.image.ref.name'                 annotation.   | <a href="https://bazel.build/docs/build-ref.html#labels">Label</a> | optional | None |
| <a id="oci_push-registry"></a>registry |  A registry host to push to, if not present consult the toolchain.   | String | optional | "" |
| <a id="oci_push-repository"></a>repository |  A repository to push to, if not present consult the toolchain.   | String | optional | "" |
| <a id="oci_push-stamp"></a>stamp |  Whether to encode build information into the output. Possible values:<br><br>    - <code>stamp = 1</code>: Always stamp the build information into the output, even in         [--nostamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) builds.         This setting should be avoided, since it is non-deterministic.         It potentially causes remote cache misses for the target and         any downstream actions that depend on the result.     - <code>stamp = 0</code>: Never stamp, instead replace build information by constant values.         This gives good build result caching.     - <code>stamp = -1</code>: Embedding of build information is controlled by the         [--[no]stamp](https://docs.bazel.build/versions/main/user-manual.html#flag--stamp) flag.         Stamped targets are not rebuilt unless their dependencies change.   | Integer | optional | -1 |
| <a id="oci_push-tag"></a>tag |  (optional) A tag to include in the target reference. This will not be included on child images.<br><br>                Subject to [$(location)](https://bazel.build/reference/be/make-variables#predefined_label_variables) and ["Make variable"](https://bazel.build/reference/be/make-variabmes) substitution.<br><br>                **Stamping**<br><br>                You can use values produced by the workspace status command in your tag. To do this write a script that prints key-value pairs separated by spaces, e.g.<br><br>                <pre><code>sh                 #!/usr/bin/env bash                 echo "STABLE_KEY1 VALUE1"                 echo "STABLE_KEY2 VALUE2"                 </code></pre><br><br>                You can reference these keys in <code>tag</code> using curly braces,<br><br>                <pre><code>python                 oci_push(                     name = "push",                     tag = "v1.0-{STABLE_KEY1}",                 )                 </code></pre>   | String | optional | "" |
| <a id="oci_push-x_meta_headers"></a>x_meta_headers |  (optional) A list of key/values to to be sent to the registry as headers with an X-Meta- prefix.   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |


