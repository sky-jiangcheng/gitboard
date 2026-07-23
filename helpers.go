package main

import (
	"encoding/json"
	"path/filepath"
)

// pathBase returns the last element of a path.
func pathBase(p string) string {
	return filepath.Base(p)
}

// pathDir returns all but the last element of a path.
func pathDir(p string) string {
	return filepath.Dir(p)
}

// jsonUnmarshalSafe unmarshals JSON into v, silently ignoring errors.
func jsonUnmarshalSafe(data string, v interface{}) {
	_ = json.Unmarshal([]byte(data), v)
}

// marshalJSON marshals v to JSON bytes, returning nil on error.
func marshalJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
