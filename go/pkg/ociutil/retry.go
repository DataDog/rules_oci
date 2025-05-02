package ociutil

import (
	"context"
	"log"
	"time"

	retry "github.com/sethvargo/go-retry"
)

// retry max retries for http requests
const retryMaxRetries = 2

// retry max attempts for http requests
const retryMaxAttempts = retryMaxRetries + 1

func RetryOnFailure(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithJitterPercent(20, b)
	b = retry.WithMaxRetries(retryMaxRetries, b)

	attempt := 0

	return retry.Do(
		ctx,
		b,
		func(ctx context.Context) error {
			attempt++
			if err := fn(ctx); err != nil {
				log.Printf(
					"failed retry attempt %d/%d: %v",
					attempt,
					retryMaxAttempts,
					err,
				)
				return err
			}
			return nil
		},
	)
}
