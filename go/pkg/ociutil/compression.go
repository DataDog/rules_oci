package ociutil

import (
	"fmt"
	"os"
)

type Compression int

const (
	CompressionNone Compression = iota
	CompressionGzip
	CompressionZstd
)

func DetectCompression(path string) (Compression, error) {
	f, err := os.Open(path)
	if err != nil {
		return CompressionNone, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	// Read up to 4 bytes for magic numbers
	var hdr [4]byte
	n, err := f.Read(hdr[:])
	if err != nil {
		return CompressionNone, fmt.Errorf("failed to read %s: %w", path, err)
	}

	// gzip: 1F 8B
	if n >= 2 && hdr[0] == 0x1F && hdr[1] == 0x8B {
		return CompressionGzip, nil
	}

	// zstd: 28 B5 2F FD
	if n >= 4 && hdr[0] == 0x28 && hdr[1] == 0xB5 &&
		hdr[2] == 0x2F && hdr[3] == 0xFD {
		return CompressionZstd, nil
	}

	return CompressionNone, nil
}
