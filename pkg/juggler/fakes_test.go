//nolint:lll
package juggler

import (
	"context"
	"reflect"
)

var _ Component = FakeComponent{}
var _ KeepOnUninstall = FakeComponent{}
var _ StatusVisibility = FakeComponent{}

type FakeComponent struct {
	Name          string
	Dependencies  []Component
	Enabled       bool
	HookFuncs     ComponentHooks
	Allowed       bool
	KeepInstalled bool
	Internal      bool
}

// KeepOnUninstall implements KeepOnUninstall.
func (f FakeComponent) KeepOnUninstall() bool {
	return f.KeepInstalled
}

func (f FakeComponent) IsInstallable(context.Context) (bool, error) {
	return f.Allowed, nil
}

func (f FakeComponent) GetName() string {
	if f.Name != "" {
		return f.Name
	}
	return "FakeComponent"
}

func (f FakeComponent) GetDependencies() []Component {
	return f.Dependencies
}

func (f FakeComponent) IsEnabled() bool {
	return f.Enabled
}

// Hooks implements Component.
func (f FakeComponent) Hooks() ComponentHooks {
	return f.HookFuncs
}

func (f FakeComponent) IsStatusInternal() bool {
	return f.Internal
}

// ---------------------------------------------------------------------------------------------------

var _ Component = FakeComponent2{}

type FakeComponent2 struct {
	Name         string
	Dependencies []Component
	Enabled      bool
	HookFuncs    ComponentHooks
	Allowed      bool
}

func (f FakeComponent2) IsInstallable(context.Context) (bool, error) {
	return f.Allowed, nil
}

func (f FakeComponent2) GetName() string {
	if f.Name != "" {
		return f.Name
	}
	return "FakeComponent2"
}

func (f FakeComponent2) GetDependencies() []Component {
	return f.Dependencies
}

func (f FakeComponent2) IsEnabled() bool {
	return f.Enabled
}

// Hooks implements Component.
func (f FakeComponent2) Hooks() ComponentHooks {
	return f.HookFuncs
}

// ---------------------------------------------------------------------------------------------------

var _ ComponentReconciler = FakeReconciler{}
var _ OrphanedComponentsDetector = FakeReconciler{}

type FakeReconciler struct {
	ObserverFunc                 func(ctx context.Context, component Component) (ComponentObservation, error)
	UninstallFunc                func(ctx context.Context, component Component) error
	UpdateFunc                   func(ctx context.Context, component Component) error
	InstallFunc                  func(ctx context.Context, component Component) error
	PreUninstallFunc             func(ctx context.Context, component Component) error
	PreInstallFunc               func(ctx context.Context, component Component) error
	PreUpdateFunc                func(ctx context.Context, component Component) error
	KnownTypesFunc               func() []reflect.Type
	DetectOrphanedComponentsFunc func(_ context.Context, configuredComponents []Component) ([]Component, error)
}

// DetectOrphanedComponents implements OrphanedComponentsDetector.
func (f FakeReconciler) DetectOrphanedComponents(ctx context.Context, configuredComponents []Component) ([]Component, error) {
	return f.DetectOrphanedComponentsFunc(ctx, configuredComponents)
}

// KnownTypes implements ComponentReconciler.
func (f FakeReconciler) KnownTypes() []reflect.Type {
	return f.KnownTypesFunc()
}

// PreUninstall implements juggler.ComponentReconciler.
func (f FakeReconciler) PreUninstall(ctx context.Context, component Component) error {
	if f.PreUninstallFunc == nil {
		return nil
	}
	return f.PreUninstallFunc(ctx, component)
}

// PreInstall implements juggler.ComponentReconciler.
func (f FakeReconciler) PreInstall(ctx context.Context, component Component) error {
	if f.PreInstallFunc == nil {
		return nil
	}
	return f.PreInstallFunc(ctx, component)
}

// PreUpdate implements juggler.ComponentReconciler.
func (f FakeReconciler) PreUpdate(ctx context.Context, component Component) error {
	if f.PreUpdateFunc == nil {
		return nil
	}
	return f.PreUpdateFunc(ctx, component)
}

func (f FakeReconciler) Observe(ctx context.Context, component Component) (ComponentObservation, error) {
	return f.ObserverFunc(ctx, component)
}

func (f FakeReconciler) Uninstall(ctx context.Context, component Component) error {
	return f.UninstallFunc(ctx, component)
}

func (f FakeReconciler) Update(ctx context.Context, component Component) error {
	if f.UpdateFunc == nil {
		return nil
	}
	return f.UpdateFunc(ctx, component)
}

func (f FakeReconciler) Install(ctx context.Context, component Component) error {
	return f.InstallFunc(ctx, component)
}
