<!-- Generated with Stardoc: http://skydoc.bazel.build -->

public providers

<a id="OCIDescriptor"></a>

## OCIDescriptor

<pre>
OCIDescriptor(<a href="#OCIDescriptor-file">file</a>, <a href="#OCIDescriptor-descriptor_file">descriptor_file</a>, <a href="#OCIDescriptor-artifact_type">artifact_type</a>, <a href="#OCIDescriptor-data">data</a>, <a href="#OCIDescriptor-annotations">annotations</a>, <a href="#OCIDescriptor-digest">digest</a>, <a href="#OCIDescriptor-media_type">media_type</a>, <a href="#OCIDescriptor-size">size</a>,
              <a href="#OCIDescriptor-urls">urls</a>)
</pre>

An OCI descriptor. See https://github.com/opencontainers/image-spec/blob/main/descriptor.md

**FIELDS**


| Name  | Description |
| :------------- | :------------- |
| <a id="OCIDescriptor-file"></a>file |  A file object of the content this descriptor describes    |
| <a id="OCIDescriptor-descriptor_file"></a>descriptor_file |  A file object with the information in this provider    |
| <a id="OCIDescriptor-artifact_type"></a>artifact_type |  Optional. The type of an artifact when the descriptor points to an artifact    |
| <a id="OCIDescriptor-data"></a>data |  Optional. An embedded representation of the referenced content    |
| <a id="OCIDescriptor-annotations"></a>annotations |  Optional. Arbitrary metadata for this descriptor    |
| <a id="OCIDescriptor-digest"></a>digest |  Required. The digest of the targeted content    |
| <a id="OCIDescriptor-media_type"></a>media_type |  Required. The media type of the referenced content    |
| <a id="OCIDescriptor-size"></a>size |  Required. The size, in bytes, of the raw content    |
| <a id="OCIDescriptor-urls"></a>urls |  Optional. A list of URIs from which this object MAY be downloaded    |


<a id="OCILayout"></a>

## OCILayout

<pre>
OCILayout(<a href="#OCILayout-blob_index">blob_index</a>, <a href="#OCILayout-files">files</a>)
</pre>

OCI Layout

**FIELDS**


| Name  | Description |
| :------------- | :------------- |
| <a id="OCILayout-blob_index"></a>blob_index |  -    |
| <a id="OCILayout-files"></a>files |  -    |


<a id="OCIReferenceInfo"></a>

## OCIReferenceInfo

<pre>
OCIReferenceInfo(<a href="#OCIReferenceInfo-registry">registry</a>, <a href="#OCIReferenceInfo-repository">repository</a>, <a href="#OCIReferenceInfo-tag">tag</a>, <a href="#OCIReferenceInfo-tag_file">tag_file</a>, <a href="#OCIReferenceInfo-digest">digest</a>)
</pre>

Refers to any artifact represented by an OCI-like reference URI

**FIELDS**


| Name  | Description |
| :------------- | :------------- |
| <a id="OCIReferenceInfo-registry"></a>registry |  the URI where the artifact is stored    |
| <a id="OCIReferenceInfo-repository"></a>repository |  a namespace for an artifact    |
| <a id="OCIReferenceInfo-tag"></a>tag |  a organizational reference within a repository    |
| <a id="OCIReferenceInfo-tag_file"></a>tag_file |  a file containing the organizational reference within a repository    |
| <a id="OCIReferenceInfo-digest"></a>digest |  a file containing the digest of the artifact    |


