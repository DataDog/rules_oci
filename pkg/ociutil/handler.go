package ociutil

import (
	"context"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// CopyContentHandler copies the parent descriptor from the provider to the
// ingestor
func CopyContentHandler(handler images.HandlerFunc, from content.Provider, to content.Ingester) images.HandlerFunc {
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		err := CopyContent(ctx, from, to, desc)
		if err != nil {
			return nil, err
		}

		return handler(ctx, desc)
	}
}

// CopyContentHandler copies the parent descriptor from the provider to the
// ingestor
func CopyContentFromHandler(ctx context.Context, handler images.HandlerFunc, from content.Provider, to content.Ingester, desc ocispec.Descriptor) error {
	children, err := handler.Handle(ctx, desc)
	if err != nil {
		return err
	}

	for _, child := range children {
		err = CopyContentFromHandler(ctx, handler, from, to, child)
		if err != nil {
			return err
		}
	}

	err = CopyContent(ctx, from, to, desc)
	if err != nil {
		return err
	}

	return nil
}

// ContentTypesFilterHandler filters the children of the handler to only include
// the listed content types
func ContentTypesFilterHandler(handler images.HandlerFunc, contentTypes ...string) images.HandlerFunc {
	set := make(stringSet)
	set.Add(contentTypes...)
	return func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		children, err := handler(ctx, desc)
		if err != nil {
			return nil, err
		}

		var rtChildren []ocispec.Descriptor
		for _, c := range children {
			if set.Contains(c.MediaType) {
				rtChildren = append(rtChildren, c)
			}
		}

		return rtChildren, nil
	}
}

// StringSet is a set of strings, used to check the existance of strings,
// this can be replaced once generics are introduced in Go 1.18
type stringSet map[string]bool

// Add add a variable list of strings to the set
func (ss stringSet) Add(strs ...string) {
	for _, st := range strs {
		ss[st] = true
	}
}

// Contains checks if a string is in the set, if it is return true, false
// otherwise.
func (ss stringSet) Contains(str string) bool {
	_, ok := ss[str]
	return ok
}
