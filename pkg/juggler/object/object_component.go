package object

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

// ObjectComponent is an interface for manageable components, specifically for plain client.Objects.
type ObjectComponent interface {
	juggler.Component

	// BuildObjectToReconcile returns an empty API object of the type that the ObjectComponent represents.
	// The object's desired state must be reconciled with the existing state inside the ReconcileObject(...) function.
	// It is called regardless of creating or updating an object.
	// Any information (also in `metadata`, e.g. labels) will be overridden before ReconcileObject(...) is called.
	BuildObjectToReconcile(ctx context.Context) (client.Object, types.NamespacedName, error)

	// ReconcileObject brings the client.Object closer to the desired state.
	ReconcileObject(ctx context.Context, obj client.Object) error

	// IsObjectHealthy returns if the object is healthy.
	IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness
}

type FilterCriteria []client.ListOption

type ConvertFunc func(list client.ObjectList) []juggler.Component

type SameFunc func(configured, detected juggler.Component) bool

type DetectorContext struct {
	// FilterCriteria describes a list of Options which identify all objects which can
	// potentially become orphaned. This should exclude objects created by the end-user
	FilterCriteria FilterCriteria
	// ConvertFunc is a transformation-func which converts a list of orphaned objects
	// into a list of known juggler components
	ConvertFunc ConvertFunc
	// SameFunc is a comparison-func which allows to compare two juggler components
	SameFunc SameFunc
	// ListType describes the type the applied object should have inside the cluster
	ListType client.ObjectList
}

// OrphanedObjectsDetector describes an interface for handling orphaned resources.
// It should be implemented by MCP components which can leave orphaned resources after
// said component is being removed from the MCP
type OrphanedObjectsDetector interface {
	OrphanDetectorContext() DetectorContext
}

func HasComponentLabel() client.ListOption {
	return client.HasLabels{labelComponentName}
}
