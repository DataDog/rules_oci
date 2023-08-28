package deb2layer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/blakesmith/ar"
)

const (
	// dpkgStatusDir is the directory to store the control file for dpkg
	// to recognize the package.
	dpkgStatusDir = "/var/lib/dpkg/status.d"

	// pkgMetadataFile is the name of the metadata file within the control
	// section
	pkgMetadataFile = "control"

	debHeader            = "debian-binary"
	debControlFilePrefix = "control"
	debDataFilePrefix    = "data"
)

var (
	// ErrDuplicateSection is returned when there is a duplicate deb package
	// sections
	ErrDuplicateSection = fmt.Errorf("duplicate section")

	// ErrSectionNotFound is returned when an expected section is not found
	ErrSectionNotFound = fmt.Errorf("section not found")
)

// DebToLayer convernts a deb package into a tar file that can be used as a
// layer in an OCI image. Including adding the appropriate files for dpkg to
// track the package.
//
// This approach has a few limitations:
//   - We don't pay attention to the metadata in the control file, including
//     the additional dependencies that may be decalared or checking that the
//     architecture is correct.
//   - Any package mantainer scripts will not be run, which breaks some packages
//     entirely.
func DebToLayer(debReader io.Reader, writer io.Writer) error {
	var foundData, foundControl, foundDeb bool
	arReader := ar.NewReader(debReader)
	tarWriter := tar.NewWriter(writer)

	for {
		header, err := arReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// > System V ar uses a '/' character (0x2F) to mark the end of the filename;
		// https://en.wikipedia.org/wiki/Ar_(Unix)
		name := header.Name
		if strings.HasSuffix(name, "/") {
			name = name[:len(name)-1]
		}

		if strings.HasPrefix(name, debControlFilePrefix) {
			if foundControl {
				return fmt.Errorf("%w: %q", ErrDuplicateSection, debControlFilePrefix)
			}

			reader, err := extToReader(getFullExt(name), arReader)
			if err != nil {
				return err
			}

			tarReader, ok := reader.(*tar.Reader)
			if !ok {
				return fmt.Errorf("%q section is not a tar file: %q (%T)", debControlFilePrefix, name, reader)
			}

			err = appendControlToTar(tarReader, tarWriter)
			if err != nil {
				return err
			}

			foundControl = true
		} else if strings.HasPrefix(name, debDataFilePrefix) {
			if foundData {
				return fmt.Errorf("%w: %q", ErrDuplicateSection, debDataFilePrefix)
			}

			reader, err := extToReader(getFullExt(name), arReader)
			if err != nil {
				return err
			}

			tarReader, ok := reader.(*tar.Reader)
			if !ok {
				return fmt.Errorf("%q section is not a tar file: %q (%T)", debDataFilePrefix, name, reader)
			}

			// XXX potentially optimize to just io.Copy the data over, rather
			// than copy through the tar interface
			err = copyTar(tarWriter, tarReader)
			if err != nil {
				return fmt.Errorf("failed to append data from deb: %w", err)
			}

			foundData = true
		} else if name == debHeader {
			if foundDeb {
				return fmt.Errorf("%w: %q", ErrDuplicateSection, debHeader)
			}

			foundDeb = true
		} else {
			return fmt.Errorf("unknown entry in .deb: %v", name)
		}
	}

	return nil
}

// CopyFileFromDeb copies a file from the debian file to a writer.
func CopyFileFromDeb(name string, debReader io.Reader, writer io.Writer) error {
	arReader := ar.NewReader(debReader)
	for {
		header, err := arReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// > System V ar uses a '/' character (0x2F) to mark the end of the filename;
		// https://en.wikipedia.org/wiki/Ar_(Unix)
		name := header.Name
		if strings.HasSuffix(name, "/") {
			name = name[:len(name)-1]
		}

		if strings.HasPrefix(name, debDataFilePrefix) {

			reader, err := extToReader(getFullExt(name), arReader)
			if err != nil {
				return err
			}

			tarReader, ok := reader.(*tar.Reader)
			if !ok {
				return fmt.Errorf("%q section is not a tar file: %q (%T)", debDataFilePrefix, name, reader)
			}

			_, err = seekToTarFile(tarReader, name)
			if err != nil {
				return err
			}

			_, err = io.Copy(writer, tarReader)
			if err != nil {
				return err
			}

			return nil
		}
	}

	return ErrSectionNotFound
}

// seekToTarFile seeks forward to the tar entry whose header's name is
// equivalent to the one provider. If found, the provided *tar.Reader will be
// left at the start of the data, otherwise it will leave at io.EOF.
func seekToTarFile(tr *tar.Reader, name string) (*tar.Header, error) {
	// clean the paths as there is usually an excess "./"
	cleanTargetName := filepath.Clean(name)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		if filepath.Clean(header.Name) == cleanTargetName {
			return header, nil
		}
	}

	return nil, fmt.Errorf("no file named %q", name)
}

// copyTar copies from a tar reader onto a tar writer.
//
// This function isn't really necessary normally as you can just squash two tars
// together, but it is needed when you don't have the underlying
// io.Reader/Writers.
func copyTar(tw *tar.Writer, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, tr)
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	// packageNameReg is used to strip out the package name from a Debian
	// package's control file.
	packageNameReg = regexp.MustCompile(`Package:\s*(?P<pkg_name>\w+).*`)
)

// getPackageName gets the package name from a control file.
func getPackageName(data []byte) string {
	matches := packageNameReg.FindSubmatch(data)
	if matches == nil {
		return ""
	}

	nameMatches := matches[packageNameReg.SubexpIndex("pkg_name")]
	return string(nameMatches)
}

// appendControlToTar finds the control file from the control.tar[.gz] archive
// and copies it into the dpkg tracking directory.
func appendControlToTar(tr *tar.Reader, tw *tar.Writer) error {
	header, err := seekToTarFile(tr, pkgMetadataFile)
	if err != nil {
		return err
	}

	var controlBuf bytes.Buffer
	_, err = io.Copy(&controlBuf, tr)
	if err != nil {
		return err
	}

	controlData := (&controlBuf).Bytes()

	name := getPackageName(controlData)
	if name == "" {
		return fmt.Errorf("no name in control file")
	}

	// Write the control file within the dpkg direction with the name being that
	// of the package.
	err = tw.WriteHeader(&tar.Header{
		Name: filepath.Join(dpkgStatusDir, name),
		Size: header.Size,
	})
	if err != nil {
		return err
	}

	_, err = io.CopyN(tw, bytes.NewReader(controlData), header.Size)
	if err != nil {
		return err
	}

	err = tw.Flush()
	if err != nil {
		return err
	}

	return nil
}

// getFullExt gets the "full" extension of a file, aka start the extension at
// the first index of '.', rather than the last like filepath.Ext.
func getFullExt(path string) string {
	base := filepath.Base(path)

	idx := strings.IndexByte(base, '.')
	if idx < 0 {
		return ""
	}

	return base[idx:]
}

// trimLastExtension trims the last extension off of a string.
func trimLastExtension(ext string) string {
	idx := strings.LastIndexByte(ext, '.')
	if idx < 0 {
		return ""
	}

	return ext[:idx]
}

// extToReader converts a reader into the specific reader that is required based
// on the file extension. For example, if you pass .tar.gz, you'll get a gzip
// reader wrapped with a tar reader.
func extToReader(ext string, inReader io.Reader) (io.Reader, error) {
	var err error
	var outReader io.Reader

	switch filepath.Ext(ext) {
	case "":
		return inReader, nil
	case ".gz":
		outReader, err = gzip.NewReader(inReader)
		if err != nil {
			return nil, err
		}
	case ".tar":
		outReader = tar.NewReader(inReader)
	default:
		return nil, fmt.Errorf("unknown extension %q", ext)
	}

	return extToReader(trimLastExtension(ext), outReader)
}
