package ociutil

import (
	"time"

	retry "github.com/sethvargo/go-retry"
)

// retry backoff strategy for http requests
var retryBackoffStrategy retry.Backoff

// retry max attempts for http requests
const retryMaxAttempts = retryMaxRetries + 1

// retry max retries for http requests
const retryMaxRetries = 2

func init() {
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithJitterPercent(20, b)
	b = retry.WithMaxRetries(retryMaxRetries, b)
	retryBackoffStrategy = b
}
