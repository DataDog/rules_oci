// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package tarutil

import (
	"archive/tar"
	"io"
	"os"
	"time"
)

// AppendFileToTarWriter appends a file (given as a filepath) to a tarfile
// through the tarfile interface.
func AppendFileToTarWriter(filePath string, loc string, tw *tar.Writer) error {
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

	hdr.ChangeTime = time.Time{}
	hdr.ModTime = time.Time{}
	hdr.AccessTime = time.Time{}

	hdr.Name = loc

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}
