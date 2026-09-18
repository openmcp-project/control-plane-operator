package utils

import (
	"testing"

	"gotest.tools/v3/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

const (
	testFoo   = "foo"
	testBar   = "bar"
	testItem  = "item"
	testCount = "count"
)

func TestSetLabel(t *testing.T) {
	tests := []struct {
		name  string
		obj   v1.Object
		label string
		value string
		want  map[string]string
	}{
		{
			name:  "add new label to object",
			obj:   &v1beta1.ControlPlane{},
			label: testFoo,
			value: testBar,
			want:  map[string]string{testFoo: testBar},
		},
		{
			name:  "update existing label",
			obj:   &v1beta1.ControlPlane{ObjectMeta: v1.ObjectMeta{Labels: map[string]string{testFoo: testBar}}},
			label: testFoo,
			value: "baz",
			want:  map[string]string{testFoo: "baz"},
		},
		{
			name:  "add a second label to object",
			obj:   &v1beta1.ControlPlane{ObjectMeta: v1.ObjectMeta{Labels: map[string]string{testFoo: testBar}}},
			label: "abc",
			value: "xyz",
			want:  map[string]string{testFoo: testBar, "abc": "xyz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLabel(tt.obj, tt.label, tt.value)
			assert.DeepEqual(t, tt.obj.GetLabels(), tt.want)
		})
	}
}

func TestSetManagedBy(t *testing.T) {
	tests := []struct {
		name string
		obj  v1.Object
		want map[string]string
	}{
		{
			name: "set managed by label",
			obj:  &v1beta1.ControlPlane{},
			want: map[string]string{LabelManagedBy: LabelManagedByValue},
		},
		{
			name: "update existing label",
			obj: &v1beta1.ControlPlane{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{LabelManagedBy: testFoo},
				},
			},
			want: map[string]string{LabelManagedBy: LabelManagedByValue},
		},
		{
			name: "add a second label to object",
			obj:  &v1beta1.ControlPlane{ObjectMeta: v1.ObjectMeta{Labels: map[string]string{testFoo: testBar}}},
			want: map[string]string{testFoo: testBar, LabelManagedBy: LabelManagedByValue},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetManagedBy(tt.obj)
			assert.DeepEqual(t, tt.obj.GetLabels(), tt.want)
		})
	}
}

func TestIsManaged(t *testing.T) {
	got := IsManaged()
	assert.DeepEqual(t, got, client.MatchingLabels{LabelManagedBy: LabelManagedByValue})
}

func TestHasComponentLabel(t *testing.T) {
	got := HasComponentLabel()
	assert.DeepEqual(t, got, client.HasLabels{LabelComponentName})
}
