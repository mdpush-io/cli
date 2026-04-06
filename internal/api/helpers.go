package api

import "encoding/json"

// jsonUnmarshal is a thin wrapper so docs.go can parse raw bytes without importing encoding/json.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
