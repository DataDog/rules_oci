package ociutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	dref "github.com/containerd/containerd/reference/docker"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// CopyJSONToFileAndCreateDescriptor encodes inf to json and then writes it to a
// file, returning the descriptor.
func CopyJSONToFileAndCreateDescriptor(inf interface{}, outFile string) (ocispec.Descriptor, error) {
	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(inf)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return CopyReaderToFileAndCreateDescriptor(&buf, outFile)
}

// CopyReaderToFileAndCreateDescriptor copys a reader to a file and then returns
// a descriptor.
func CopyReaderToFileAndCreateDescriptor(reader io.Reader, outFile string) (ocispec.Descriptor, error) {
	f, err := os.Create(outFile)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	defer f.Close()

	return CopyAndCreateDescriptor(reader, f)
}

// CopyAndCreateDescriptor copys a reader to a writer and returns a descriptor,
// note that this desciptor will only have the Digest and Size fields populated.
func CopyAndCreateDescriptor(reader io.Reader, writer io.Writer) (ocispec.Descriptor, error) {
	digester := digest.SHA256.Digester()
	n, err := io.Copy(io.MultiWriter(writer, digester.Hash()), reader)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return ocispec.Descriptor{
		Digest: digester.Digest(),
		Size:   n,
	}, nil
}

// WriteDescriptorToFile writes an OCI descriptor to a file.
func WriteDescriptorToFile(path string, desc ocispec.Descriptor) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("couldn't create descriptor: %w", err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(&desc)
	if err != nil {
		return fmt.Errorf("couldn't encode descriptor: %w", err)
	}

	return nil
}

// DescriptorFromFile reads an OCI descriptor from a file path.
//
// XXX Descriptor must be json encoded
func ReadDescriptorFromFile(path string) (ocispec.Descriptor, error) {
	f, err := os.Open(path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("couldn't read descriptor: %w", err)
	}
	defer f.Close()

	var desc ocispec.Descriptor
	err = json.NewDecoder(f).Decode(&desc)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("couldn't parse descriptor: %w", err)
	}

	return desc, nil
}

// DescriptortoURL converts a combination of a registry and a descriptor to a
// URL that the blob can be downloaded from.
func DescriptorToURL(reg string, desc ocispec.Descriptor) (string, error) {
	n, err := NamedRef(reg)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%v/v2/%v/blobs/%v", dref.Domain(n), dref.Path(n), desc.Digest), nil
}
