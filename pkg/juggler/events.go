package juggler

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

type EventType string

const (
	// Information only and will not cause any problems
	EventNormal EventType = EventType(corev1.EventTypeNormal)

	// These events are to warn that something might go wrong
	EventWarning EventType = EventType(corev1.EventTypeWarning)
)

type ComponentEventType string

const (
	// No event
	ComponentEventNone ComponentEventType = ""

	// Information only and will not cause any problems
	ComponentEventNormal ComponentEventType = ComponentEventType(EventNormal)

	// These events are to warn that something might go wrong
	ComponentEventWarning ComponentEventType = ComponentEventType(EventWarning)
)

// EventRecorder is an opinionated record.EventRecorder for the Juggler.
type EventRecorder interface {
	Event(eventtype EventType, reason, message string)

	// Eventf is just like Event, but with Sprintf for the message field.
	Eventf(eventtype EventType, reason, messageFmt string, args ...interface{})

	// AnnotatedEventf is just like eventf, but with annotations attached
	AnnotatedEventf(annotations map[string]string, eventtype EventType, reason, messageFmt string, args ...interface{})
}

// NewEventRecorder returns a new (opinionated) EventRecorder for a given record.EventRecorder and a runtime.Object.
func NewEventRecorder(recorder record.EventRecorder, object runtime.Object) EventRecorder {
	return &ObjectEventRecorder{
		recorder: recorder,
		object:   object,
	}
}

// ObjectEventRecorder is the EventRecorder for the Juggler. It encapsulates the record.EventRecorder interface.
type ObjectEventRecorder struct {
	recorder record.EventRecorder
	object   runtime.Object
}

func (oer *ObjectEventRecorder) Event(eventtype EventType, reason, message string) {
	oer.recorder.Event(oer.object, string(eventtype), reason, message)
}

func (oer *ObjectEventRecorder) Eventf(eventtype EventType, reason, messageFmt string, args ...interface{}) {
	oer.recorder.Eventf(oer.object, string(eventtype), reason, messageFmt, args...)
}

func (oer *ObjectEventRecorder) AnnotatedEventf(annotations map[string]string, eventtype EventType, reason,
	messageFmt string, args ...interface{}) {

	oer.recorder.AnnotatedEventf(oer.object, annotations, string(eventtype), reason, messageFmt, args...)
}

// ComponentEventRecorder returns a new (opinionated) EventRecorder for a given EventRecorder and a Component.
func NewComponentEventRecorder(recorder EventRecorder, component Component) *ComponentEventRecorder {
	return &ComponentEventRecorder{
		recorder:  recorder,
		component: component,
	}
}

type ComponentEventRecorder struct {
	recorder  EventRecorder
	component Component
}

func (c *ComponentEventRecorder) Event(status ComponentStatus, message string) {
	if status.EmitsEvent == ComponentEventNone {
		return
	}
	c.recorder.Event(EventType(status.EmitsEvent), c.component.GetName()+status.Name, message)
}
