<!-- Generated with Stardoc: http://skydoc.bazel.build -->



<a id="#oci_image"></a>

## oci_image

<pre>
oci_image(<a href="#oci_image-name">name</a>, <a href="#oci_image-annotations">annotations</a>, <a href="#oci_image-arch">arch</a>, <a href="#oci_image-base">base</a>, <a href="#oci_image-entrypoint">entrypoint</a>, <a href="#oci_image-layers">layers</a>, <a href="#oci_image-os">os</a>)
</pre>


    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image-annotations"></a>annotations |     | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |
| <a id="oci_image-arch"></a>arch |     | String | optional | "" |
| <a id="oci_image-base"></a>base |     | <a href="https://bazel.build/docs/build-ref.html#labels">Label</a> | required |  |
| <a id="oci_image-entrypoint"></a>entrypoint |  -   | List of strings | optional | [] |
| <a id="oci_image-layers"></a>layers |     | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="oci_image-os"></a>os |     | String | optional | "" |


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


    

**ATTRIBUTES**


| Name  | Description | Type | Mandatory | Default |
| :------------- | :------------- | :------------- | :------------- | :------------- |
| <a id="oci_image_layer-name"></a>name |  A unique name for this target.   | <a href="https://bazel.build/docs/build-ref.html#name">Name</a> | required |  |
| <a id="oci_image_layer-directory"></a>directory |     | String | optional | "" |
| <a id="oci_image_layer-file_map"></a>file_map |  -   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: Label -> String</a> | optional | {} |
| <a id="oci_image_layer-files"></a>files |     | <a href="https://bazel.build/docs/build-ref.html#labels">List of labels</a> | optional | [] |
| <a id="oci_image_layer-symlinks"></a>symlinks |     | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |


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
oci_push(<a href="#oci_push-name">name</a>, <a href="#oci_push-headers">headers</a>, <a href="#oci_push-manifest">manifest</a>, <a href="#oci_push-registry">registry</a>, <a href="#oci_push-repository">repository</a>, <a href="#oci_push-tag">tag</a>, <a href="#oci_push-x_meta_headers">x_meta_headers</a>)
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
| <a id="oci_push-tag"></a>tag |  (optional) A tag to include in the target reference. This will not be included on child images."   | String | optional | "" |
| <a id="oci_push-x_meta_headers"></a>x_meta_headers |  (optional) A list of key/values to to be sent to the registry as headers with an X-Meta- prefix.   | <a href="https://bazel.build/docs/skylark/lib/dict.html">Dictionary: String -> String</a> | optional | {} |


