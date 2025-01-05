use tracing_subscriber::EnvFilter;

/// Initializes logging
pub(crate) fn init_logging() {
    let _ = tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .try_init();
}

pub(crate) mod fs {
    use std::io;
    use std::path::Path;

    use anyhow::Context as _;
    use fs_err as fs;

    /// Copies the file at src to dst
    ///
    /// Note: We use this instead of fs::copy to avoid copying file permissions
    pub(crate) fn copy<P, Q>(src: P, dst: Q) -> Result<(), anyhow::Error>
    where
        P: AsRef<Path>,
        Q: AsRef<Path>,
    {
        let src = src.as_ref();
        let dst = dst.as_ref();
        let mut reader = io::BufReader::new(fs::File::open(src)?);
        let mut writer = io::BufWriter::new(fs::File::create(dst)?);
        io::copy(&mut reader, &mut writer)
            .with_context(|| format!("Failed to copy {} to {}", src.display(), dst.display()))?;
        Ok(())
    }
}
