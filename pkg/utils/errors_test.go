package utils

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func TestIsCRDNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no error",
			err:  nil,
			want: false,
		},
		{
			name: "no kind match error",
			err:  &meta.NoKindMatchError{},
			want: true,
		},
		{
			name: "no resource match error",
			err:  &meta.NoResourceMatchError{},
			want: true,
		},
		{
			name: "resource discovery failed error",
			err:  &apiutil.ErrResourceDiscoveryFailed{},
			want: true,
		},
		{
			name: "not found error",
			err: &apiutil.ErrResourceDiscoveryFailed{
				schema.GroupVersion{}: errors.NewNotFound(schema.GroupResource{}, ""),
			},
			want: true,
		},
		{
			name: "multiple not found error",
			err: &apiutil.ErrResourceDiscoveryFailed{
				schema.GroupVersion{}: errors.NewNotFound(schema.GroupResource{}, ""),
				schema.GroupVersion{}: errors.NewNotFound(schema.GroupResource{}, ""),
			},
			want: true,
		},
		{
			name: "Not found and forbidden error",
			err: &apiutil.ErrResourceDiscoveryFailed{
				schema.GroupVersion{}: errors.NewNotFound(schema.GroupResource{}, ""),
				schema.GroupVersion{}: errors.NewForbidden(schema.GroupResource{}, "", nil),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCRDNotFound(tt.err); got != tt.want {
				t.Errorf("IsCRDNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
