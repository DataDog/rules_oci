package tarutil

import (
	"archive/tar"
	"io"
	"os"
	"time"
)

// AppendFileToTarWriter appends a file (given as a filepath) to a tarfile
// through the tarfile interface.
func AppendFileToTarWriter(
	filePath string,
	loc string,
	mode int64,
	uname,
	gname string,
	tw *tar.Writer,
) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}

	hdr.AccessTime = time.Time{}
	hdr.ChangeTime = time.Time{}
	hdr.Gname = gname
	hdr.ModTime = time.Time{}
	hdr.Mode = mode
	hdr.Uname = uname

	hdr.Name = loc

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}
