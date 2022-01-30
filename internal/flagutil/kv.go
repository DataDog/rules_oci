package flagutil

import (
	"fmt"
	"strings"
)

var (
	ErrInvalidFlagMap = fmt.Errorf("not a valid mapping, must be k=k")
)

type KeyValueFlag struct {
	Map map[string]string
}

func (k *KeyValueFlag) String() string {
	return ""
}

func (k *KeyValueFlag) Set(value string) error {
	if k.Map == nil {
		k.Map = make(map[string]string)
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) < 2 {
		return fmt.Errorf("%w: %v", ErrInvalidFlagMap, value)
	}

	k.Map[parts[0]] = parts[1]
	return nil
}
