package utils

import "encoding/json"

// MustMarshal returns the JSON encoding of v and panics if there is an error.
func MustMarshal(v any) []byte {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return jsonBytes
}
