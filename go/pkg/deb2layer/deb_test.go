package deb2layer

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Rebuild deb with `dpkg-deb -Zgzip --build testdeb`

var (
	expectedFiles = []string{"/var/lib/dpkg/status.d/testdeb", "./", "./deb_text.txt"}
)

// TestDebToLayer checks that the expected files are in the resulting layer
// based on a deb file created by Bazel's rules_pkg
func TestDebToLayer(t *testing.T) {
	// XXX Fix generating deb package
	f, err := os.Open("testdeb.deb")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	// I wanted to io.Pipe, but too lazy to send the errors back via
	// a channel. TIL that you can't t.Fail on a different goroutine
	var buf bytes.Buffer
	err = DebToLayer(f, &buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	var actualFiles []string
	tarReader := tar.NewReader(&buf)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("%v", err)
		}

		actualFiles = append(actualFiles, header.Name)
	}

	f.Close()

	assert.Equal(t, expectedFiles, actualFiles)
}

func TestCopyFileFromDeb(t *testing.T) {
	f, err := os.Open("testdeb.deb")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	err = CopyFileFromDeb("deb_text.txt", f, &buf)
	if err != nil {
		t.Fatalf("%v", err)
	}

	assert.Equal(t, "my-deb\n", buf.String())
}
