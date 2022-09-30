package jsonutil

import (
	"encoding/json"
	"os"
)

// DecodeFromFile is a helper to read json from a file into a struct easily.
func DecodeFromFile(filePath string, inf interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewDecoder(file).Decode(inf)
}
