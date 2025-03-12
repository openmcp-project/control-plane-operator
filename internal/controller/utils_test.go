package controller

import (
	"context"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/stretchr/testify/assert"
)

func Test_shortenToXCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "short",
			expected: "short",
			maxLen:   100,
		},
		{
			name:     "long string",
			input:    "this-is-a-very-a-very-a-very-long-string-that-is-over-63-characters",
			expected: "this-is-a-very-a-very-a-very-long-string-that-is-over--4b610537",
			maxLen:   63,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := shortenToXCharacters(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, actual)
			assert.LessOrEqual(t, len(actual), tt.maxLen)
		})
	}
}

func newContext() context.Context {
	ctx := context.Background()
	ctx = log.IntoContext(ctx, log.Log)
	return ctx
}

func newRequest(obj client.Object) ctrl.Request {
	return ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(obj),
	}
}
