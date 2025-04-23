package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/DataDog/rules_oci/go/internal/flagutil"
	"github.com/DataDog/rules_oci/go/pkg/ociutil"

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/images"
	"github.com/urfave/cli/v2"
)

// calcDelayForNextRetry calculates the delay to wait for a given retry attempt _after_ the current attempt.
func calcDelayForNextRetry(currentAttempt int, initialDelay time.Duration, maxJitter time.Duration) time.Duration {
	currentAttempt++
	delay := time.Duration(currentAttempt) * initialDelay

	var jitter time.Duration
	if maxJitter > 0 {
		jitter = time.Duration(rand.Int63n(int64(maxJitter)))
	}

	return delay + jitter
}

type RetryConfig struct {
	maxAttempts  int
	initialDelay time.Duration
	maxJitter    time.Duration
}

// Retry retries a function until it succeeds or the context is done.
func Retry(ctx context.Context, config RetryConfig, fn func() error) error {
	ticker := time.NewTicker(0)
	defer ticker.Stop()

	var errs []error

loop:
	for attempt := range config.maxAttempts {
		err := fn()
		if err == nil {
			return nil
		}
		errs = append(errs, err)

		// on the last attempt we return the error right away rather than waiting to return
		if attempt == config.maxAttempts-1 {
			break
		}

		delay := calcDelayForNextRetry(attempt, config.initialDelay, config.maxJitter)
		ticker.Reset(delay)
		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			break loop
		case <-ticker.C:
		}
	}

	return errors.Join(errs...)
}

func PushCmd(c *cli.Context) error {
	localProviders, err := LoadLocalProviders(c.StringSlice("layout"), c.String("layout-relative"))
	if err != nil {
		return err
	}

	allProviders := ociutil.MultiProvider(localProviders...)

	baseDesc, err := ociutil.ReadDescriptorFromFile(c.String("desc"))
	if err != nil {
		return fmt.Errorf("failed to read base descriptor: %w", err)
	}

	headers := c.Generic("headers").(*flagutil.KeyValueFlag).Map
	if headers == nil {
		headers = map[string]string{}
	}
	// tack on the X-Meta- prefix
	for k, v := range c.Generic("x_meta_headers").(*flagutil.KeyValueFlag).Map {
		headers["X-Meta-"+k] = v
	}

	resolver := ociutil.ResolverWithHeaders(headers)

	ref := c.String("target-ref")

	pusher, err := resolver.Pusher(c.Context, ref)
	if err != nil {
		return fmt.Errorf("failed to create pusher: %w", err)
	}

	regIng, ok := pusher.(content.Ingester)
	if !ok {
		return fmt.Errorf("pusher not an ingester: %T", pusher)
	}

	// take care of copying any children first
	imagesHandler := images.ChildrenHandler(allProviders)
	err = ociutil.CopyChildrenFromHandler(c.Context, imagesHandler, allProviders, regIng, baseDesc)
	if err != nil {
		return fmt.Errorf("failed to push child content to registry: %w", err)
	}

	// if a tag exists, use it for the parent
	tag := c.String("parent-tag")
	if tag != "" {
		ref = ref + ":" + tag
		pusher, err = resolver.Pusher(c.Context, ref)
		if err != nil {
			return fmt.Errorf("failed to create parent pusher: %w", err)
		}

		regIng, ok = pusher.(content.Ingester)
		if !ok {
			return fmt.Errorf("parent pusher not an ingester: %T", pusher)
		}
	}

	// push the parent last (in case of image index)
	rcfg := RetryConfig{
		maxAttempts:  5,
		initialDelay: 1 * time.Second,
		maxJitter:    1 * time.Second,
	}
	err = Retry(c.Context, rcfg, func() error {
		return ociutil.CopyContent(c.Context, allProviders, regIng, baseDesc)
	})
	if err != nil {
		return fmt.Errorf("failed to push parent content to registry: %w", err)
	}

	fmt.Printf("Reference: %v@%v\n", ref, baseDesc.Digest.String())

	return nil
}
