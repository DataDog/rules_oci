// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/remotes"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// FetchAndJSONDecodefetches a the content from a descriptor
// and unmarshalls it into a struct
func FetchAndJSONDecode(ctx context.Context, fetch remotes.FetcherFunc, desc ocispec.Descriptor, inf interface{}) error {
	reader, err := fetch(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = json.NewDecoder(reader).Decode(inf)
	if err != nil {
		return err
	}

	return nil
}

// ProviderJSONDecode fetches content content from a provider and decodes
// the json content.
func ProviderJSONDecode(ctx context.Context, provider content.Provider, desc ocispec.Descriptor, inf interface{}) error {
	reader, err := provider.ReaderAt(ctx, desc)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = json.NewDecoder(io.NewSectionReader(reader, 0, desc.Size)).Decode(inf)
	if err != nil {
		return err
	}

	return nil
}

// IngestorJSONEncode encodes json and saves it to a ingester.
func IngestorJSONEncode(ctx context.Context, ing content.Ingester, mediaType string, inf interface{}) (ocispec.Descriptor, error) {
	data, err := json.Marshal(inf)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("unable to marshal JSON: %w", err)
	}
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Size:      int64(len(data)),
		Digest:    digest.SHA256.FromBytes(data),
	}

	writer, err := ing.Writer(ctx, content.WithDescriptor(desc))
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	err = content.Copy(ctx, writer, bytes.NewReader(data), desc.Size, desc.Digest)
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return desc, nil
}
