package juggler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type test_externalImplementType struct {
	FakeComponent
}

var _ StatusVisibility = test_externalImplementType{}
var _ Component = test_externalImplementType{}

func (c test_externalImplementType) IsStatusInternal() bool {
	return false
}

type test_internalImplementType struct {
	FakeComponent
}

var _ StatusVisibility = test_internalImplementType{}

func (c test_internalImplementType) IsStatusInternal() bool {
	return true
}

func Test_isComponentInternal(t *testing.T) {
	// externalImplement implements StatusVisibility but reports
	// external
	externalImplement := test_externalImplementType{}

	if isComponentInternal(externalImplement) {
		t.Errorf("externalImplement should be considered as external")
	}

	// internalImplement implements StatusVisibility and reports
	// internal
	internalImplement := test_internalImplementType{}

	if !isComponentInternal(internalImplement) {
		t.Errorf("internalImplement should be considered as internal")
	}

	// FakeComponent has an Internal field that works
	fakeInternalComponent := FakeComponent{Internal: true}
	if !isComponentInternal(fakeInternalComponent) {
		t.Errorf("fakeInternalComponent should be considered as internal")
	}

}

func TestComponentResult_ToCondition(t *testing.T) {
	type fields struct {
		Component Component
		Result    ComponentStatus
		Message   string
	}
	tests := []struct {
		name   string
		fields fields
		want   v1.Condition
	}{
		{
			name: "errored component maps to failing condition",
			fields: fields{
				Component: FakeComponent{Name: "Foo"},
				Result:    ComponentStatus{Name: "FailedStatus", IsReady: false},
				Message:   "boom",
			},
			want: v1.Condition{
				Type:    "FooReady",
				Status:  "False",
				Reason:  "FailedStatus",
				Message: "boom",
			},
		},
		{
			name: "No error component maps to good condition",
			fields: fields{
				Component: FakeComponent{Name: "Foo"},
				Result:    ComponentStatus{Name: "GoodStatus", IsReady: true},
			},
			want: v1.Condition{
				Type:   "FooReady",
				Status: "True",
				Reason: "GoodStatus",
			},
		},
		{
			name: "No result returns unknown condition",
			fields: fields{
				Component: FakeComponent{Name: "Foo"},
				Result:    ComponentStatus{IsReady: false},
			},
			want: v1.Condition{
				Type:   "FooReady",
				Status: "False",
				Reason: "Unknown",
			},
		},
		{
			name: "Internal component's type starts with lowercase letter",
			fields: fields{
				Component: FakeComponent{Name: "Foo", Internal: true},
				Result:    ComponentStatus{IsReady: false},
			},
			want: v1.Condition{
				Type:   "fooReady",
				Status: "False",
				Reason: "Unknown",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ComponentResult{
				Component: tt.fields.Component,
				Result:    tt.fields.Result,
				Message:   tt.fields.Message,
			}

			is := r.ToCondition()
			assert.Equalf(t, tt.want.Type, is.Type, "ToCondition()")
			assert.Equalf(t, tt.want.Reason, is.Reason, "ToCondition()")
			assert.Equalf(t, tt.want.Message, is.Message, "ToCondition()")
			assert.Equalf(t, tt.want.Status, is.Status, "ToCondition()")
		})
	}
}
