package juggler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"

	"github.com/openmcp-project/control-plane-operator/api/v1beta1"
)

func TestObjectEventRecorder_Event(t *testing.T) {
	cp := v1beta1.ControlPlane{}
	recorder := record.NewFakeRecorder(1)
	fer := ObjectEventRecorder{object: &cp, recorder: recorder}
	fer.Event("FakeType", "FakeReason", "FakeMessage")

	expected := []string{"FakeType FakeReason FakeMessage"}
	c := time.After(wait.ForeverTestTimeout)
	for _, e := range expected {
		select {
		case a := <-recorder.Events:
			assert.Equal(t, e, a)
		case <-c:
			t.Errorf("Expected event %q, got nothing", e)
			// continue iterating to print all expected events
		}
	}
}

func TestObjectEventRecorder_Eventf(t *testing.T) {
	cp := v1beta1.ControlPlane{}
	recorder := record.NewFakeRecorder(1)
	fer := ObjectEventRecorder{object: &cp, recorder: recorder}
	fer.Eventf("FakeType", "FakeReason", "FakeMessage with %s", "FakeAddition")

	expected := []string{"FakeType FakeReason FakeMessage with FakeAddition"}
	c := time.After(wait.ForeverTestTimeout)
	for _, e := range expected {
		select {
		case a := <-recorder.Events:
			assert.Equal(t, e, a)
		case <-c:
			t.Errorf("Expected event %q, got nothing", e)
		}
	}
}
