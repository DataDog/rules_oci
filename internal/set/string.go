// Unless explicitly stated otherwise all files in this repository are licensed under the MIT License.
//
// This product includes software developed at Datadog (https://www.datadoghq.com/). Copyright 2021 Datadog, Inc.

package set

// String is a set of strings, used to check the existance of strings,
// this can be replaced once generics are introduced in Go 1.18
type String map[string]bool

// Add add a variable list of strings to the set
func (ss String) Add(strs ...string) {
	for _, st := range strs {
		ss[st] = true
	}
}

// Contains checks if a string is in the set, if it is return true, false
// otherwise.
func (ss String) Contains(str string) bool {
	_, ok := ss[str]
	return ok
}
