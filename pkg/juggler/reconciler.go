package juggler

import (
	"context"
	"reflect"
)

type ComponentReconciler interface {
	// KnownTypes returns a list of component types that are supported by this reconciler.
	KnownTypes() []reflect.Type
	// Observe returns a ComponentObservation to return the state of a Component
	// inside the cluster to the ComponentReconciler
	Observe(ctx context.Context, component Component) (ComponentObservation, error)
	// Uninstall triggers an uninstall of a Component in the cluster
	Uninstall(ctx context.Context, component Component) error
	// Update updates a Component in the cluster
	Update(ctx context.Context, component Component) error
	// Install triggers an install of a Component in the cluster
	Install(ctx context.Context, component Component) error
	// PreUninstall calls the pre-uninstall hook of a Component.
	PreUninstall(ctx context.Context, component Component) error
	// PreUninstall calls the pre-install hook of a Component.
	PreInstall(ctx context.Context, component Component) error
	// PreUpdate calls the pre-update hook of a Component.
	PreUpdate(ctx context.Context, component Component) error
}

type ComponentObservation struct {
	ResourceExists bool
	ResourceHealthiness
}

type ResourceHealthiness struct {
	Healthy bool
	Message string
}

// OrphanedComponentsDetector can be implemented by a `ComponentReconciler` to signal that it
// supports the discovery of orphaned components.
type OrphanedComponentsDetector interface {
	// DetectOrphanedComponents searches for orphaned components. The `configuredComponents` contains a list of
	// configured components supported by the reconciler. It should be used to calculate the delta between
	// existing and configured components.
	// Orphaned=Existing\Configured (set difference D=M\N).
	DetectOrphanedComponents(ctx context.Context, configuredComponents []Component) ([]Component, error)
}
