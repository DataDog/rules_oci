use std::collections::HashMap;
use std::io::{self, IsTerminal};
use std::path::{Path, PathBuf};
use std::process::{Command, Stdio};

use anyhow::Context as _;
use colored::Colorize as _;
use fs_err as fs;
use oci_spec::image::{Digest, Platform};

pub(crate) fn oci_load(
    platforms_path: PathBuf,
    repository: String,
    tar_path: PathBuf,
) -> Result<(), anyhow::Error> {
    docker_load(&tar_path)?;

    let platforms = {
        let s = fs::read_to_string(&platforms_path)?;
        let platforms =
            serde_json::from_str::<HashMap<Digest, Platform>>(&s).with_context(|| {
                format!(
                    "Failed to deserialize {} into a valid HashMap<Digest, Platform>",
                    platforms_path.display()
                )
            })?;
        if platforms.is_empty() {
            anyhow::bail!("No platforms found in {}", platforms_path.display());
        }
        platforms
    };

    let image_name = match validate_image_name(&repository) {
        Some(image_name) => image_name,
        None => {
            let msg = format!(
                concat!(
                    "WARN: Loaded image(s) into the docker daemon, but failed to retag it(them) ",
                    "with the image name '{}' because that name contains invalid characters. ",
                    "Docker only allows the following characters in image names: ['a..=z', ",
                    "'0..=9', '.', '_', '-', '/']. You can still access the image(s) via ",
                    "its(their) digest(s) above",
                ),
                repository,
            );
            if io::stderr().is_terminal() {
                eprintln!("{}", msg.yellow());
            } else {
                eprintln!("{}", msg);
            };
            return Ok(());
        }
    };

    let is_multi_arch = platforms.len() > 1;
    for (digest, platform) in &platforms {
        let old_id = digest.to_string();
        let new_id = if is_multi_arch {
            format!("{}:latest-{}", image_name, platform.architecture())
        } else {
            format!("{}:latest", image_name)
        };
        docker_tag(&old_id, &new_id)?;
    }

    Ok(())
}

fn docker_load(tar_path: &Path) -> Result<(), anyhow::Error> {
    let cmd = "docker load";

    let msg = format!("{} < {}", cmd, tar_path.display());
    if io::stdout().is_terminal() {
        println!("{}", msg.blue());
    } else {
        println!("{}", msg);
    }

    let mut child = Command::new("bash")
        .arg("-c")
        .arg(cmd)
        .stdin(Stdio::piped())
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .spawn()
        .with_context(|| format!("Failed to spawn command '{}'", cmd))?;

    {
        let mut tar_reader = io::BufReader::new(fs::File::open(tar_path)?);
        let mut stdin = child.stdin.take().expect("cannot fail");
        io::copy(&mut tar_reader, &mut stdin)
            .with_context(|| format!("Failed to copy {} to stdin", tar_path.display()))?;
        // Note: stdin is dropped here
    }

    let status = child
        .wait()
        .with_context(|| format!("Failed to wait on command '{}'", cmd))?;

    if !status.success() {
        anyhow::bail!(
            "Command '{}'. Exit code: {}",
            cmd,
            status.code().unwrap_or(-1),
        );
    }

    Ok(())
}

fn docker_tag(old_id: &str, new_id: &str) -> Result<(), anyhow::Error> {
    let cmd = format!("docker tag {} {}", old_id, new_id);

    if io::stdout().is_terminal() {
        println!("{}", cmd.blue());
    } else {
        println!("{}", cmd);
    }

    let mut child = Command::new("bash")
        .arg("-c")
        .arg(&cmd)
        .stdout(Stdio::inherit())
        .stderr(Stdio::inherit())
        .spawn()
        .with_context(|| format!("Failed to spawn command '{}'", cmd))?;

    let status = child
        .wait()
        .with_context(|| format!("Failed to wait on command '{}'", cmd))?;

    if !status.success() {
        anyhow::bail!(
            "Command '{}'. Exit code: {}",
            cmd,
            status.code().unwrap_or(-1),
        );
    }

    Ok(())
}

fn validate_image_name(name: &str) -> Option<String> {
    name.chars()
        .map(|c| match c {
            'a'..='z' | '0'..='9' | '.' | '_' | '-' | '/' => Some(c),
            'A'..='Z' => Some(c.to_ascii_lowercase()),
            _ => None,
        })
        .collect()
}
