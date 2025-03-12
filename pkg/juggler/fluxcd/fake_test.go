package fluxcd

import (
	"context"
	"reflect"

	"github.com/openmcp-project/control-plane-operator/pkg/juggler"
)

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

var _ FluxComponent = FakeFluxComponent{}

type FakeFluxComponent struct {
	BuildSourceRepositoryFunc  func(ctx context.Context) (SourceAdapter, error)
	BuildManifestoFunc         func(ctx context.Context) (Manifesto, error)
	IsAllowedToBeInstalledFunc func(ctx context.Context) bool
	GetNameFunc                string
	GetDependenciesFunc        []juggler.Component
	IsEnabledFunc              bool
	HookFunc                   juggler.ComponentHooks
}

func (f FakeFluxComponent) BuildSourceRepository(ctx context.Context) (SourceAdapter, error) {
	return f.BuildSourceRepositoryFunc(ctx)
}

func (f FakeFluxComponent) BuildManifesto(ctx context.Context) (Manifesto, error) {
	return f.BuildManifestoFunc(ctx)
}

func (f FakeFluxComponent) IsInstallable(ctx context.Context) (bool, error) {
	return f.IsAllowedToBeInstalledFunc(ctx), nil
}

func (f FakeFluxComponent) GetName() string {
	return f.GetNameFunc
}

func (f FakeFluxComponent) GetDependencies() []juggler.Component {
	return f.GetDependenciesFunc
}

func (f FakeFluxComponent) IsEnabled() bool {
	return f.IsEnabledFunc
}

// Hooks implements Component.
func (f FakeFluxComponent) Hooks() juggler.ComponentHooks {
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
