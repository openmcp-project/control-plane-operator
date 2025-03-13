package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SetNested(t *testing.T) {
	testCases := []struct {
		desc        string
		m           map[string]any
		v           any
		path        []string
		expected    map[string]any
		expectedErr error
	}{
		{
			desc:        "should fail when map is nil",
			expectedErr: ErrMapNil,
		},
		{
			desc:        "should fail when map no field path is provided",
			m:           map[string]any{},
			expected:    map[string]any{},
			expectedErr: ErrNoPath,
		},
		{
			desc: "should fail when field path leads to wrong map type",
			m: map[string]any{
				"item": map[string]int{
					"count": 1,
				},
			},
			path: []string{"item", "count"},
			expected: map[string]any{
				"item": map[string]int{
					"count": 1,
				},
			},
			expectedErr: ErrNotAStringAnyMap,
		},
		{
			desc: "should not override nested field",
			m: map[string]any{
				"item": map[string]any{
					"count": 1,
				},
			},
			path: []string{"item", "count"},
			v:    2,
			expected: map[string]any{
				"item": map[string]any{
					"count": 1,
				},
			},
			expectedErr: nil,
		},
		{
			desc: "should set nested field",
			m: map[string]any{
				"item": map[string]any{},
			},
			path: []string{"item", "count"},
			v:    2,
			expected: map[string]any{
				"item": map[string]any{
					"count": 2,
				},
			},
			expectedErr: nil,
		},
		{
			desc: "should create sub-map and set nested field",
			m:    map[string]any{},
			path: []string{"item", "count"},
			v:    2,
			expected: map[string]any{
				"item": map[string]any{
					"count": 2,
				},
			},
			expectedErr: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			actualErr := SetNestedDefault(tC.m, tC.v, tC.path...)
			assert.Equal(t, tC.expected, tC.m)
			assert.Equal(t, tC.expectedErr, actualErr)
		})
	}
}

func Test_GetNestedValue(t *testing.T) {
	testCases := []struct {
		desc        string
		m           map[string]any
		path        []string
		expected    any
		expectedErr error
	}{
		{
			desc:        "should return error when map is nil",
			expectedErr: ErrMapNil,
		},
		{
			desc:        "should return error when path is empty",
			m:           map[string]any{},
			expectedErr: ErrNoPath,
		},
		{
			desc:        "should return error when value was not found",
			m:           map[string]any{},
			path:        []string{"item"},
			expectedErr: ErrValueNotFound,
		},
		{
			desc:        "should return error when value was not found because nested map is missing",
			m:           map[string]any{},
			path:        []string{"item", "count"},
			expectedErr: ErrValueNotFound,
		},
		{
			desc: "should return error when value was not found in nested map",
			m: map[string]any{
				"item": map[string]any{},
			},
			path:        []string{"item", "count"},
			expectedErr: ErrValueNotFound,
		},
		{
			desc: "should fail when field path leads to wrong map type",
			m: map[string]any{
				"item": map[string]int{
					"count": 1,
				},
			},
			path:        []string{"item", "count"},
			expectedErr: ErrNotAStringAnyMap,
		},
		{
			desc: "should find value in nested map",
			m: map[string]any{
				"item": map[string]any{
					"count": 1,
				},
			},
			path:        []string{"item", "count"},
			expected:    1,
			expectedErr: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			v, err := GetNestedValue(tC.m, tC.path...)
			assert.Equal(t, tC.expected, v)
			assert.Equal(t, tC.expectedErr, err)
		})
	}
}
