package fluxcd

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

type FluxComponent interface {
	juggler.Component

	BuildSourceRepository(ctx context.Context) (SourceAdapter, error)
	BuildManifesto(ctx context.Context) (Manifesto, error)
}

type FluxResource interface {
	GetObject() client.Object
	GetObjectKey() client.ObjectKey
	GetHealthiness() juggler.ResourceHealthiness

	Reconcile(desired FluxResource) error
	ApplyDefaults()
}

type SourceAdapter interface {
	FluxResource

	// Empty returns a new adapter that wraps the same type.
	// It contains an empty (non-nil) client.Object retaining only the name and namespace.
	// This is useful when working with controllerutil.CreateOrUpdate(...).
	Empty() SourceAdapter
}

type Manifesto interface {
	FluxResource

	// Empty returns a new adapter that wraps the same type.
	// It contains an empty (non-nil) client.Object retaining only the name and namespace.
	// This is useful when working with controllerutil.CreateOrUpdate(...).
	Empty() Manifesto
}
