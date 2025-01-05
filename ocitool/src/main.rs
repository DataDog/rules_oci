mod cmd;
mod utils;

use std::io::{self, IsTerminal};
use std::path::PathBuf;

use clap::{Parser, Subcommand};
use colored::Colorize as _;

/// A tool to be called from bazel rules that works with OCI images
#[derive(Debug, Parser)]
struct Args {
    #[command(subcommand)]
    cmd: Cmd,
}

#[derive(Debug, Subcommand)]
enum Cmd {
    /// Creates an OCI layout directory from a descriptor file and a set of files
    OciDir {
        /// Path to the json-encoded descriptor file for the index or manifest
        #[arg(long)]
        descriptor_path: PathBuf,

        /// Path to a file to potentially include in the blobs directory
        #[arg(long = "file")]
        files: Vec<PathBuf>,

        /// Path to the output directory
        #[arg(long)]
        out_dir: PathBuf,

        /// Path to the output platforms.json file
        #[arg(long)]
        out_platforms_path: PathBuf,
    },
    /// Load an OCI index into the docker daemon
    OciLoad {
        /// Path to a json file containin a map of Digest->Platform for the
        /// manifests in the index
        #[arg(long)]
        platforms_path: PathBuf,

        /// The name of the repository to tag the loaded images with
        #[arg(long)]
        repository: String,

        /// Path to the tarball containing the OCI index
        #[arg(long)]
        tar_path: PathBuf,
    },
}

fn main() {
    if let Err(e) = run() {
        let msg = format!("Error: {}", e);
        if io::stderr().is_terminal() {
            eprintln!("{}", msg.red());
        } else {
            eprintln!("{}", msg);
        }
        std::process::exit(1);
    }
}

fn run() -> Result<(), anyhow::Error> {
    let Args { cmd } = Args::parse();

    utils::init_logging();

    match cmd {
        Cmd::OciDir {
            descriptor_path,
            files,
            out_dir,
            out_platforms_path,
        } => cmd::oci_dir(descriptor_path, files, out_dir, out_platforms_path),
        Cmd::OciLoad {
            platforms_path,
            repository,
            tar_path,
        } => cmd::oci_load(platforms_path, repository, tar_path),
    }
}
