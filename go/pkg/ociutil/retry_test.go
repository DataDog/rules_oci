package ociutil

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestRetryOnFailure(t *testing.T) {
	ctx := context.Background()

	// Should not produce an error
	var count int32 = 0
	err := RetryOnFailure(ctx, func(ctx context.Context) error {
		c := atomic.AddInt32(&count, 1)
		if c < retryMaxAttempts {
			return errors.New("I failed!")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Should produce an error
	count = 0
	err = RetryOnFailure(ctx, func(ctx context.Context) error {
		c := atomic.AddInt32(&count, 1)
		if c < retryMaxAttempts+1 {
			return errors.New("I failed!")
		}
		return nil
	})
	if err == nil {
		t.Fatalf("expected error, got %v", err)
	}
}
