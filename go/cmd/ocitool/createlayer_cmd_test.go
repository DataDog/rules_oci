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
		{input: "0x755", expected: 0x755},
		{input: "1877", expected: 0x755},
		{input: "0b11101010101", expected: 0x755},
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
