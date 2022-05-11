// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package ociutil

import (
	"io"
	"sync/atomic"
)

// WriterCounter is counter for io.Writer, doesn't prevent concurrent writes.
type WriterCounter struct {
	io.Writer
	count uint64
}

// NewWriterCounter function create new WriterCounter
func NewWriterCounter(w io.Writer) *WriterCounter {
	return &WriterCounter{
		Writer: w,
	}
}

func (counter *WriterCounter) Write(buf []byte) (int, error) {
	n, err := counter.Writer.Write(buf)
	atomic.AddUint64(&counter.count, uint64(n))
	return n, err
}

// Count function return counted bytes
func (counter *WriterCounter) Count() uint64 {
	return atomic.LoadUint64(&counter.count)
}
