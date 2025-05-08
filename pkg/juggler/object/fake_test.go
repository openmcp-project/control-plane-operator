package object

import (
	"context"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

var fakeFilterLabel = "fake.object.component/managed"

var _ ObjectComponent = FakeObjectComponent{}
var _ OrphanedObjectsDetector = FakeObjectComponent{}

type FakeObjectComponent struct {
	allowedToBeInstalled       bool
	name                       string
	dependencies               []juggler.Component
	enabled                    bool
	hooks                      juggler.ComponentHooks
	BuildObjectToReconcileFunc func(ctx context.Context) (client.Object, types.NamespacedName, error)
	ReconcileObjectFunc        func(ctx context.Context, obj client.Object) error
	IsObjectHealthyFunc        func(obj client.Object) juggler.ResourceHealthiness
}

// BuildObjectToReconcile implements ObjectComponent.
func (f FakeObjectComponent) BuildObjectToReconcile(ctx context.Context) (client.Object, types.NamespacedName, error) {
	return f.BuildObjectToReconcileFunc(ctx)
}

// ReconcileObject implements ObjectComponent.
func (f FakeObjectComponent) ReconcileObject(ctx context.Context, obj client.Object) error {
	return f.ReconcileObjectFunc(ctx, obj)
}

// OrphanDetectorContext implements OrphanedObjectsDetector.
func (f FakeObjectComponent) OrphanDetectorContext() DetectorContext {
	return DetectorContext{
		ListType: &corev1.ConfigMapList{},
		FilterCriteria: FilterCriteria{
			client.HasLabels{fakeFilterLabel},
		},
		ConvertFunc: func(list client.ObjectList) []juggler.Component {
			comps := []juggler.Component{}
			for _, cm := range (list.(*corev1.ConfigMapList)).Items {
				comps = append(comps, FakeObjectComponent{name: cm.Name})
			}
			return comps
		},
		SameFunc: func(configured, detected juggler.Component) bool {
			return strings.EqualFold(configured.GetName(), detected.GetName())
		},
	}
}

func (f FakeObjectComponent) IsInstallable(context.Context) (bool, error) {
	return f.allowedToBeInstalled, nil
}

func (f FakeObjectComponent) GetName() string {
	return f.name
}

func (f FakeObjectComponent) GetDependencies() []juggler.Component {
	return f.dependencies
}

func (f FakeObjectComponent) IsEnabled() bool {
	return f.enabled
}

// Hooks implements Component.
func (f FakeObjectComponent) Hooks() juggler.ComponentHooks {
	return f.hooks
}

func (f FakeObjectComponent) IsObjectHealthy(obj client.Object) juggler.ResourceHealthiness {
	return f.IsObjectHealthyFunc(obj)
}

// ---------------------------------------------------------------------------------------------------

var _ juggler.Component = FakeComponent{}

type FakeComponent struct {
	IsAllowedToBeInstalledFunc func(ctx context.Context) bool
	GetNameFunc                string
	GetDependenciesFunc        []juggler.Component
	IsEnabledFunc              bool
	HookFunc                   juggler.ComponentHooks
}

func (f FakeComponent) IsInstallable(ctx context.Context) (bool, error) {
	return f.IsAllowedToBeInstalledFunc(ctx), nil
}

func (f FakeComponent) GetName() string {
	return f.GetNameFunc
}

func (f FakeComponent) GetDependencies() []juggler.Component {
	return f.GetDependenciesFunc
}

func (f FakeComponent) IsEnabled() bool {
	return f.IsEnabledFunc
}

// Hooks implements Component.
func (f FakeComponent) Hooks() juggler.ComponentHooks {
	return f.HookFunc
}

// ---------------------------------------------------------------------------------------------------

var _ juggler.ComponentReconciler = FakeReconciler{}

type FakeReconciler struct {
	ObserverFunc     func(ctx context.Context, component juggler.Component) (juggler.ComponentObservation, error)
	UninstallFunc    func(ctx context.Context, component juggler.Component) error
	UpdateFunc       func(ctx context.Context, component juggler.Component) error
	InstallFunc      func(ctx context.Context, component juggler.Component) error
	PreUninstallFunc func(ctx context.Context, component juggler.Component) error
	PreInstallFunc   func(ctx context.Context, component juggler.Component) error
	PreUpdateFunc    func(ctx context.Context, component juggler.Component) error
	KnownTypesFunc   func() []reflect.Type
}

// KnownTypes implements juggler.ComponentReconciler.
func (f FakeReconciler) KnownTypes() []reflect.Type {
	return f.KnownTypesFunc()
}

// PreUninstall implements juggler.ComponentReconciler.
func (f FakeReconciler) PreUninstall(ctx context.Context, component juggler.Component) error {
	if f.PreUninstallFunc == nil {
		return nil
	}
	return f.PreUninstallFunc(ctx, component)
}

// PreInstall implements juggler.ComponentReconciler.
func (f FakeReconciler) PreInstall(ctx context.Context, component juggler.Component) error {
	if f.PreInstallFunc == nil {
		return nil
	}
	return f.PreInstallFunc(ctx, component)
}

// PreUpdate implements juggler.ComponentReconciler.
func (f FakeReconciler) PreUpdate(ctx context.Context, component juggler.Component) error {
	if f.PreUpdateFunc == nil {
		return nil
	}
	return f.PreUpdateFunc(ctx, component)
}

//nolint:lll
func (f FakeReconciler) Observe(ctx context.Context, component juggler.Component) (juggler.ComponentObservation, error) {
	return f.ObserverFunc(ctx, component)
}

func (f FakeReconciler) Uninstall(ctx context.Context, component juggler.Component) error {
	return f.UninstallFunc(ctx, component)
}

func (f FakeReconciler) Update(ctx context.Context, component juggler.Component) error {
	return f.UpdateFunc(ctx, component)
}

func (f FakeReconciler) Install(ctx context.Context, component juggler.Component) error {
	return f.InstallFunc(ctx, component)
}
