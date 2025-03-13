package utils

import (
	"testing"
)

func TestMustMarshal(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{
			name: "nil",
			v:    nil,
			want: "null",
		},
		{
			name: "string",
			v:    "foo",
			want: `"foo"`,
		},
		{
			name: "int",
			v:    1,
			want: "1",
		},
		{
			name: "struct",
			v:    struct{ Foo string }{Foo: "bar"},
			want: `{"Foo":"bar"}`,
		},
		{
			name: "slice",
			v:    []string{"foo", "bar"},
			want: `["foo","bar"]`,
		},
		{
			name: "map",
			v:    map[string]string{"foo": "bar"},
			want: `{"foo":"bar"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MustMarshal(tt.v)

			if len(got) == 0 {
				t.Errorf("MustMarshal() = %v, want non-empty", got)
			}
			if string(got) != tt.want {
				t.Errorf("MustMarshal() = %v, want %v", got, tt.want)
			}
		})
	}
}
