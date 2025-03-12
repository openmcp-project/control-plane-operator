package utils

import "errors"

var (
	ErrNotAStringAnyMap = errors.New("sub-map is not of type map[string]any")
	ErrNoPath           = errors.New("at least one path element needs to be specified")
	ErrMapNil           = errors.New("map is nil")
	ErrValueNotFound    = errors.New("value not found")
)

// SetNestedDefault traverses a map and sets a default value if no map entry exists yet under the given key.
func SetNestedDefault(m map[string]any, v any, path ...string) error {
	if m == nil {
		return ErrMapNil
	}
	if len(path) == 0 {
		return ErrNoPath
	}
	if len(path) == 1 {
		// Value set already, don't override
		if _, ok := m[path[0]]; ok {
			return nil
		}
		// Set default
		m[path[0]] = v
		return nil
	}

	if m[path[0]] == nil {
		m[path[0]] = map[string]any{}
	}

	subMap, ok := m[path[0]].(map[string]any)
	if !ok {
		return ErrNotAStringAnyMap
	}
	return SetNestedDefault(subMap, v, path[1:]...)
}

// GetNestedValue traverses a map and looks for a value under the given path.
// If the value was not found, `ErrValueNotFound` will be returned.
func GetNestedValue(m map[string]any, path ...string) (any, error) {
	if m == nil {
		return nil, ErrMapNil
	}
	if len(path) == 0 {
		return nil, ErrNoPath
	}
	if len(path) == 1 {
		// Value set, return it
		if val, ok := m[path[0]]; ok {
			return val, nil
		}
		return nil, ErrValueNotFound
	}
	if m[path[0]] == nil {
		return nil, ErrValueNotFound
	}

	subMap, ok := m[path[0]].(map[string]any)
	if !ok {
		return nil, ErrNotAStringAnyMap
	}
	return GetNestedValue(subMap, path[1:]...)
}
