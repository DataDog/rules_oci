package flagutil

import (
	"fmt"
	"strings"
)

var (
	ErrInvalidFlagMap = fmt.Errorf("not a valid mapping, must be k=k")
)

type KeyValue struct {
	Key   string
	Value string
}

// KeyValueFlag is a cli.Generic implementation that parses flag values of the form k=v
type KeyValueFlag struct {
	// List contains all the k=v values in order encountered
	List []KeyValue
	// Map contains all the k=v values as a map
	Map map[string]string
}

func (k *KeyValueFlag) String() string {
	return ""
}

func (k *KeyValueFlag) Set(value string) error {
	if k.Map == nil {
		k.Map = make(map[string]string)
	}
	if k.List == nil {
		k.List = make([]KeyValue, 0)
	}

	parts := strings.SplitN(value, "=", 2)
	if len(parts) < 2 {
		return fmt.Errorf("%w: %v", ErrInvalidFlagMap, value)
	}

	k.List = append(k.List, KeyValue{Key: parts[0], Value: parts[1]})
	k.Map[parts[0]] = parts[1]
	return nil
}
