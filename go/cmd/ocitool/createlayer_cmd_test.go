package main

import (
	"strconv"
	"testing"
)

func TestParseIntWorksAsExpected(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected int64
	}{
		{input: "0o755", expected: 0o755},
		{input: "0b111101101", expected: 0o755},
		{input: "493", expected: 0o755},
		{input: "0x1ed", expected: 0o755},
	} {
		actual, err := strconv.ParseInt(tc.input, 0, 64)
		if err != nil {
			t.Errorf("ParseInt(%q, 0, 64) unexpectedly returned an error. Error: %w", tc.input, err)
		}
		if actual != tc.expected {
			t.Errorf(
				"ParseInt(%q, 0, 64) = %d, but expected %d",
				tc.input,
				actual,
				tc.expected,
			)
		}
	}
}
