use std::collections::HashMap;
use std::io;
use std::path::Path;

use anyhow::Context as _;
use fs_err as fs;
use oci_spec::image::{Descriptor, Digest, MediaType, Platform};
use serde::Deserialize;

/// Adds an extra method to the external oci_spec::image::Descriptor type
pub(crate) trait DescriptorExt: Sized {
    fn from_path<P>(path: P) -> Result<Self, anyhow::Error>
    where
        P: AsRef<Path>;
}

impl DescriptorExt for Descriptor {
    /// Constructs a Descriptor by deserializing the file at the provided path
    fn from_path<P>(path: P) -> Result<Self, anyhow::Error>
    where
        P: AsRef<Path>,
    {
        let path = path.as_ref();

        let raw = {
            let reader = io::BufReader::new(fs::File::open(path)?);
            serde_json::from_reader::<_, DescriptorRaw>(reader).with_context(|| {
                format!(
                    "Failed to deserialize {} into a valid Descriptor",
                    path.display()
                )
            })?
        };

        let mut errors = Vec::new();
        if raw.digest.is_none() {
            errors.push("digest");
        }
        if raw.media_type.is_none() {
            errors.push("media_type");
        }
        if raw.size.is_none() {
            errors.push("size");
        }
        if !errors.is_empty() {
            anyhow::bail!(
                "Descriptor is missing required fields: {}",
                errors.join(", ")
            );
        }

        let digest = raw.digest.expect("checked above");
        let media_type = raw.media_type.expect("checked above");
        let size = raw.size.expect("checked above");

        let supported_media_types = [MediaType::ImageIndex, MediaType::ImageManifest];
        if !supported_media_types.contains(&media_type) {
            anyhow::bail!(
                "Invalid media type in descriptor. Got {}. Expected one of: {}",
                media_type,
                supported_media_types
                    .iter()
                    .map(|mt| mt.to_string())
                    .collect::<Vec<_>>()
                    .join(", ")
            );
        }

        let mut this = oci_spec::image::Descriptor::new(media_type, size, digest);
        this.set_artifact_type(raw.artifact_type);
        this.set_annotations(raw.annotations);
        this.set_data(raw.data);
        this.set_platform(raw.platform);
        this.set_urls(raw.urls);

        Ok(this)
    }
}

#[derive(Clone, Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
struct DescriptorRaw {
    #[serde(default)]
    annotations: Option<HashMap<String, String>>,

    #[serde(default)]
    artifact_type: Option<MediaType>,

    #[serde(default)]
    data: Option<String>,

    #[serde(default)]
    digest: Option<Digest>,

    #[serde(default)]
    media_type: Option<MediaType>,

    #[serde(default)]
    platform: Option<Platform>,

    #[serde(default)]
    size: Option<u64>,

    #[serde(default)]
    urls: Option<Vec<String>>,
}
