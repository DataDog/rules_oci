mod descriptor;
mod digest;

use std::collections::HashMap;
use std::io;
use std::path::{Path, PathBuf};

use anyhow::Context as _;
use fs_err as fs;
use oci_spec::image::{
    Descriptor, Digest, ImageConfiguration, ImageIndex, ImageIndexBuilder, ImageManifest,
    MediaType, Platform, SCHEMA_VERSION,
};

use crate::cmd::oci_dir::descriptor::DescriptorExt as _;
use crate::cmd::oci_dir::digest::DigestExt as _;
use crate::utils;

pub(crate) fn oci_dir(
    descriptor_path: PathBuf,
    files: Vec<PathBuf>,
    out_dir: PathBuf,
    out_platforms_path: PathBuf,
) -> Result<(), anyhow::Error> {
    // Deserialize the descriptor provided by the user
    let descriptor = Descriptor::from_path(&descriptor_path)?;

    // Create a map of Digest->Path for all 'files' provided by the user, e.g.
    // {
    //     Digest("sha256:9c03df60186916e2ab82ec082c4c841409e0399d66647acf72343b3865c68139"): "bazel-out/darwin_arm64-fastbuild/bin/examples/rust/image.layout.json",
    //     Digest("sha256:5c7ce68406b22986555d3c0922c75b60bcec49796addd09f86d9557d1897f76d"): "external/_main~_repo_rules~ubuntu_noble/blobs/sha256/5c7ce68406b22986555d3c0922c75b60bcec49796addd09f86d9557d1897f76d",
    // }
    let files_map = files
        .into_iter()
        .map(|path| Digest::calculate_from_path(&path).map(|digest| (digest, path)))
        .collect::<Result<HashMap<_, _>, _>>()?;

    // Construct the an OCI layout directory (the output)...

    // An OCI layout directory should look like this:
    // ./oci-layout
    // ./index.json
    // ./blobs/sha256/9c03df60186916e2ab82ec082c4c841409e0399d66647acf72343b3865c68139
    // ./blobs/sha256/5c7ce68406b22986555d3c0922c75b60bcec49796addd09f86d9557d1897f76d
    // ./blobs/sha256/...
    //
    // where...
    // - ./oci-layout is a file containing the json '{"imageLayoutVersion": "1.0.0"}'
    // - ./index.json is a file containing the json representation of the index
    // - ./blobs/sha256/... contains all the manifests, configs, and layers pointed to by the
    //   index, as well as the index itself
    //
    // See https://github.com/opencontainers/image-spec/blob/main/image-layout.md

    // Make the blobs/sha256 directory
    fs::create_dir_all(out_dir.join("blobs").join("sha256"))?;

    // Write the oci-layout file
    fs::write(
        out_dir.join("oci-layout"),
        br#"{"imageLayoutVersion": "1.0.0"}"#,
    )?;

    // Handle the rest differently depending on if the user provided an index or a manifest
    let mut out_platforms = HashMap::<Digest, Platform>::new();
    match descriptor.media_type() {
        MediaType::ImageIndex => {
            // Find the path of the index in the files_map
            let index_path = files_map.get(descriptor.digest()).ok_or_else(|| {
                anyhow::anyhow!("Failed to find index with digest {}", descriptor.digest())
            })?;

            // Copy the index to index.json
            utils::fs::copy(index_path, out_dir.join("index.json"))?;

            // Copy the index to the blobs directory
            utils::fs::copy(index_path, out_dir.join(descriptor.digest().path()))?;

            // Deserialize the index
            let index = {
                let reader = io::BufReader::new(fs::File::open(index_path)?);
                serde_json::from_reader::<_, ImageIndex>(reader)?
            };

            // Write all the manifests, configs, and layers to the blobs directory
            for manifest_descriptor in index.manifests() {
                write_manifest_and_associated_config_and_associated_layers_to_blobs_directory(
                    &files_map,
                    manifest_descriptor,
                    &out_dir,
                    &mut out_platforms,
                )?;
            }
        }
        MediaType::ImageManifest => {
            // Create an index
            let index = ImageIndexBuilder::default()
                .manifests(vec![descriptor])
                .media_type(MediaType::ImageIndex)
                .schema_version(SCHEMA_VERSION)
                // Note: You are also allowed to set 'annotations', 'artifactType', and 'subject'
                // on an index, but we don't do that here
                .build()?;

            // Write the index to index.json
            let index_str = serde_json::to_string_pretty(&index)?;
            fs::write(out_dir.join("index.json"), index_str.as_bytes())?;

            // Write the index to the blobs directory
            let index_digest = Digest::calculate_from_str(&index_str)?;
            fs::write(out_dir.join(index_digest.path()), index_str.as_bytes())?;

            // Write the manifest, config, and all the layers to the blobs directory
            for manifest_descriptor in index.manifests() {
                write_manifest_and_associated_config_and_associated_layers_to_blobs_directory(
                    &files_map,
                    manifest_descriptor,
                    &out_dir,
                    &mut out_platforms,
                )?;
            }
        }
        _ => {
            unreachable!(
                "checked the validity of media types in the constructor of 'descriptor' above"
            )
        }
    }

    // Write the platforms information to the out_platforms_path
    let mut writer = io::BufWriter::new(fs::File::create(out_platforms_path)?);
    serde_json::to_writer_pretty(&mut writer, &out_platforms)?;

    Ok(())
}

fn write_manifest_and_associated_config_and_associated_layers_to_blobs_directory(
    files_map: &HashMap<Digest, PathBuf>,
    manifest_descriptor: &oci_spec::image::Descriptor,
    out_dir: &Path,
    out_platforms: &mut HashMap<Digest, Platform>,
) -> Result<(), anyhow::Error> {
    // Record information about the platform of the manifest
    out_platforms.insert(
        manifest_descriptor.digest().clone(),
        manifest_descriptor.platform().clone().ok_or_else(|| {
            anyhow::anyhow!(
                "Failed to find platform for manifest with digest {}",
                manifest_descriptor.digest()
            )
        })?,
    );

    // Get the manifest path
    let manifest_path = files_map.get(manifest_descriptor.digest()).ok_or_else(|| {
        anyhow::anyhow!(
            "Failed to find manifest with digest {}",
            manifest_descriptor.digest()
        )
    })?;

    // Deserialize the manifest
    let manifest = {
        let reader = io::BufReader::new(fs::File::open(manifest_path)?);
        serde_json::from_reader::<_, ImageManifest>(reader).with_context(|| {
            format!(
                "Failed to deserialize {} into a valid ImageManifest",
                manifest_path.display()
            )
        })?
    };

    // Copy the manifest to the blobs directory
    utils::fs::copy(
        manifest_path,
        out_dir.join(manifest_descriptor.digest().path()),
    )?;

    // Get the config path
    let config_path = files_map.get(manifest.config().digest()).ok_or_else(|| {
        anyhow::anyhow!(
            "Failed to find config with digest {}",
            manifest.config().digest()
        )
    })?;

    // Deserialize the config (not required, but adds an extra check that the config is well-formed)
    let _config = {
        let reader = io::BufReader::new(fs::File::open(config_path)?);
        serde_json::from_reader::<_, ImageConfiguration>(reader).with_context(|| {
            format!(
                "Failed to deserialize {} into a valid ImageConfiguration",
                config_path.display()
            )
        })?
    };

    // Copy the config to the blobs directory
    utils::fs::copy(config_path, out_dir.join(manifest.config().digest().path()))?;

    // For each layer
    for layer_descriptor in manifest.layers() {
        // Get the layer path
        let layer_path = files_map.get(layer_descriptor.digest()).ok_or_else(|| {
            anyhow::anyhow!(
                "Failed to find layer with digest {}",
                layer_descriptor.digest()
            )
        })?;

        // Copy the layer to the blobs directory
        utils::fs::copy(layer_path, out_dir.join(layer_descriptor.digest().path()))?;
    }

    Ok(())
}
