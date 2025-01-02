<!-- Generated with Stardoc: http://skydoc.bazel.build -->

public repository rules

<a id="oci_pull"></a>

## oci_pull

<pre>
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


