use std::io;
use std::path::{Path, PathBuf};

use fs_err as fs;
use oci_spec::image::Digest;
use sha2::{Digest as _, Sha256};

/// Adds an extra method to the external oci_spec::image::Digest type
pub(crate) trait DigestExt: Sized {
    fn calculate_from_str<S>(s: S) -> Result<Self, anyhow::Error>
    where
        S: AsRef<str>;

    fn calculate_from_path<P>(path: P) -> Result<Self, anyhow::Error>
    where
        P: AsRef<Path>;

    fn path(&self) -> PathBuf;
}

impl DigestExt for Digest {
    /// Returns the digest of the provided string
    fn calculate_from_str<S>(s: S) -> Result<Self, anyhow::Error>
    where
        S: AsRef<str>,
    {
        let s = s.as_ref();
        let mut reader = s.as_bytes();
        let mut hasher = Sha256::new();
        io::copy(&mut reader, &mut hasher)?;
        let sha256 = hex::encode(hasher.finalize());
        let digest = format!("sha256:{}", sha256).parse::<Digest>()?;
        Ok(digest)
    }

    /// Returns the digest of the file at the provided path
    ///
    /// Exploits the fact that paths in the blobs directory already encode the digests of the files
    /// contained inside, so this function tries to get the digest from the file's path. Only if that
    /// fails, does it fall back to actually calculating the sha256 hash of the file
    fn calculate_from_path<P>(path: P) -> Result<Self, anyhow::Error>
    where
        P: AsRef<Path>,
    {
        let path = path.as_ref();

        let fallback = |path| {
            let mut reader = io::BufReader::new(fs::File::open(path)?);
            let mut hasher = Sha256::new();
            io::copy(&mut reader, &mut hasher)?;
            let sha256 = hex::encode(hasher.finalize());
            let digest = format!("sha256:{}", sha256).parse::<Digest>()?;
            Ok(digest)
        };

        let filename = match path
            .file_name()
            .ok_or_else(|| anyhow::anyhow!("Called .file_name() on the root of the filesystem"))?
            .to_str()
        {
            Some(filename) => filename,
            None => return fallback(path),
        };

        let parent = match path.parent().and_then(|parent| parent.to_str()) {
            Some(parent) => parent,
            None => return fallback(path),
        };

        for (algo, algo_hex_len) in [
            ("sha256", 64),  // a hex-encoded sha256 hash is 64 bytes in length
            ("sha384", 96),  //               sha384         96
            ("sha512", 128), //               sha512         128
        ] {
            if parent == algo && filename.len() == algo_hex_len {
                let digest_str = format!("{}:{}", algo, filename);
                match digest_str.parse::<Digest>() {
                    Ok(digest) => return Ok(digest),
                    Err(_) => return fallback(path),
                }
            }
        }

        fallback(path)
    }

    /// Returns the path to write a file with this digest to (relative to the OCI layout directory root)
    fn path(&self) -> PathBuf {
        Path::new("blobs")
            .join(self.algorithm().to_string())
            .join(self.digest())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_path() -> Result<(), anyhow::Error> {
        let digest = "sha256:bac5da5a7201f802fe318150094700ae1bc7b59ab093eb1abacb787049239a9e"
            .parse::<Digest>()?;
        let expected = Path::new("blobs")
            .join("sha256")
            .join("bac5da5a7201f802fe318150094700ae1bc7b59ab093eb1abacb787049239a9e");
        assert_eq!(digest.path(), expected);
        Ok(())
    }
}
