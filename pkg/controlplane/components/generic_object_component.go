package components

import (
	"context"
	"reflect"
	"strings"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
	"github.com/openmcp-project/control-plane-operator/pkg/juggler/object"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ object.ObjectComponent = &GenericObjectComponent{}
var _ TargetComponent = &GenericObjectComponent{}
var _ juggler.KeepOnUninstall = &GenericObjectComponent{}
var _ juggler.StatusVisibility = &GenericObjectComponent{}

type GenericObjectComponent struct {
	types.NamespacedName

	NameOverride        string
	TypeNameOverride    string
	Enabled             bool
	Type                client.Object
	Dependencies        []juggler.Component
	IsObjectHealthyFunc func(obj client.Object) juggler.ResourceHealthiness
	ReconcileObjectFunc func(ctx context.Context, obj client.Object) error
	KeepInstalled       bool
}

// KeepOnUninstall implements juggler.KeepOnUninstall.
func (g *GenericObjectComponent) KeepOnUninstall() bool {
	return g.KeepInstalled
}

// BuildObjectToReconcile implements object.ObjectComponent.
func (g *GenericObjectComponent) BuildObjectToReconcile(
	ctx context.Context,
) (client.Object, types.NamespacedName, error) {
	return g.Type.DeepCopyObject().(client.Object), g.NamespacedName, nil
}

// GetDependencies implements object.ObjectComponent.
func (g *GenericObjectComponent) GetDependencies() []juggler.Component {
	return g.Dependencies
}

// GetName implements object.ObjectComponent.
func (g *GenericObjectComponent) GetName() string {
	if g.NameOverride != "" {
		return g.NameOverride
	}

	parts := strings.Split(g.Name, "-")
	for i, part := range parts {
		parts[i] = cases.Title(language.English).String(part)
	}

	typeName := reflect.TypeOf(g.Type).Elem().Name()
	if g.TypeNameOverride != "" {
		typeName = g.TypeNameOverride
	}

	return typeName + strings.Join(parts, "")
}

// Hooks implements object.ObjectComponent.
func (g *GenericObjectComponent) Hooks() juggler.ComponentHooks {
	return juggler.ComponentHooks{}
}

// IsInstallable implements object.ObjectComponent.
func (g *GenericObjectComponent) IsInstallable(ctx context.Context) (bool, error) {
	return true, nil
}

// IsEnabled implements object.ObjectComponent.
func (g *GenericObjectComponent) IsEnabled() bool {
	return g.Enabled
}

// IsObjectHealthy implements object.ObjectComponent.
func (g *GenericObjectComponent) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	return g.IsObjectHealthyFunc(obj)
}

// ReconcileObject implements object.ObjectComponent.
func (g *GenericObjectComponent) ReconcileObject(ctx context.Context, obj client.Object) error {
	return g.ReconcileObjectFunc(ctx, obj)
}

// GetNamespace implements TargetComponent.
func (g *GenericObjectComponent) GetNamespace() string {
	return g.Namespace
}

// IsStatusInternal implements StatusVisibility interface.
func (g *GenericObjectComponent) IsStatusInternal() bool {
	return true
}
