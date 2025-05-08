package juggler

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
)

var (
	errNoReconcilerForComponentType = errors.New("no reconciler available for component")
	errTooManyReconcilers           = errors.New("more than one reconciler available for component")
	errDependencyNotRegistered      = errors.New("one or more dependencies are not registered")
	errDependencyNotEnabled         = errors.New("one or more dependencies are registered but not enabled")
	errHookFailed                   = errors.New("hook failed")
)

// NewJuggler initializes a new Juggler.
func NewJuggler(logger logr.Logger, recorder EventRecorder) *Juggler {
	return &Juggler{
		logger:      logger.WithName("Juggler"),
		components:  []Component{},
		reconcilers: []ComponentReconciler{},
		recorder:    recorder,
	}
}

// Juggler manages components.
type Juggler struct {
	logger      logr.Logger
	components  []Component
	reconcilers []ComponentReconciler
	recorder    EventRecorder
}

// RegisterComponent makes the Juggler aware of new components.
func (am *Juggler) RegisterComponent(component ...Component) {
	am.components = append(am.components, component...)
}

// RegisterReconciler makes the Juggler aware of new reconcilers.
func (am *Juggler) RegisterReconciler(reconciler ...ComponentReconciler) {
	am.reconcilers = append(am.reconcilers, reconciler...)
}

// RegisterOrphanedComponents calls registered reconcilers that implement the optional
// `OrphanedComponentsDetector` interface and registers orphaned components so that they can be uninstalled.
func (am *Juggler) RegisterOrphanedComponents(ctx context.Context) error {
	for _, rec := range am.reconcilers {
		if ocd, ok := rec.(OrphanedComponentsDetector); ok {
			configuredComponents := am.componentsOfReconciler(rec)
			orphaned, err := ocd.DetectOrphanedComponents(ctx, configuredComponents)
			if err != nil {
				return err
			}
			am.RegisterComponent(orphaned...)
		}
	}
	return nil
}

// RegisteredComponents returns the number of registered components.
func (am *Juggler) RegisteredComponents() int {
	return len(am.components)
}

// Reconcile compares the current and desired state for each registered
// component and takes measures to reach the desired state if necessary.
// The implementation is inspired by Crossplanes Managed resource reconciler
func (am *Juggler) Reconcile(ctx context.Context) []ComponentResult {
	results := make([]ComponentResult, 0, len(am.components))

	for _, component := range am.components {
		cr := am.reconcileComponent(ctx, component)
		NewComponentEventRecorder(am.recorder, component).Event(cr.Result, cr.Message)
		results = append(results, cr)
	}

	return results
}

func (am *Juggler) reconcileComponent(ctx context.Context, component Component) ComponentResult {
	// Find reconciler for component
	reconciler, err := am.findReconcilerFor(component)
	if err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusReconcilerNotFound,
			Message:   err.Error(),
		}
	}

	// Observe Component
	observation, err := reconciler.Observe(ctx, component)
	if err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusObservationFailed,
			Message:   err.Error(),
		}
	}

	// Resource exists, but should be uninstalled
	if observation.ResourceExists && !component.IsEnabled() {
		if kou, ok := component.(KeepOnUninstall); ok && kou.KeepOnUninstall() {
			// pretend to uninstall the component
			return ComponentResult{
				Component: component,
				Result:    StatusUninstalled,
				Message:   fmt.Sprintf("%s is marked as 'keep on uninstall'.", component.GetName()),
			}
		}

		if err := reconciler.PreUninstall(ctx, component); err != nil {
			wrappedErr := errors.Join(errHookFailed, err).Error()
			return ComponentResult{
				Component: component,
				Result:    StatusUninstallFailed,
				Message:   wrappedErr,
			}
		}

		if err := reconciler.Uninstall(ctx, component); err != nil {
			return ComponentResult{
				Component: component,
				Result:    StatusUninstallFailed,
				Message:   err.Error(),
			}
		}

		return ComponentResult{
			Component: component,
			Result:    StatusUninstalled,
			Message:   fmt.Sprintf("%s has been uninstalled successfully.", component.GetName()),
		}
	}

	// If component disabled, do nothing
	if !component.IsEnabled() {
		return ComponentResult{
			Component: component,
			Result:    StatusDisabled,
			Message:   fmt.Sprintf("%s is not enabled.", component.GetName()),
		}
	}

	is, err := component.IsInstallable(ctx)
	if err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusInstallFailed,
			Message:   fmt.Sprintf("%s not installable: %s", component.GetName(), err.Error()),
		}
	}
	if !is {
		return ComponentResult{
			Component: component,
			Result:    StatusComponentNotAllowed,
			Message:   fmt.Sprintf("%s not installable.", component.GetName()),
		}
	}

	if err := am.checkDependencies(component); err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusDependencyCheckFailed,
			Message:   err.Error(),
		}
	}

	// Resource does not exist but is enabled, then install
	if !observation.ResourceExists {
		if err := reconciler.PreInstall(ctx, component); err != nil {
			wrappedErr := errors.Join(errHookFailed, err).Error()
			return ComponentResult{
				Component: component,
				Result:    StatusInstallFailed,
				Message:   wrappedErr,
			}
		}

		err := reconciler.Install(ctx, component)
		if err != nil {
			return ComponentResult{
				Component: component,
				Result:    StatusInstallFailed,
				Message:   err.Error(),
			}
		}
		return ComponentResult{
			Component: component,
			Result:    StatusInstalled,
			Message:   fmt.Sprintf("%s has been installed successfully.", component.GetName()),
		}
	}

	if err := reconciler.PreUpdate(ctx, component); err != nil {
		wrappedErr := errors.Join(errHookFailed, err).Error()
		return ComponentResult{
			Component: component,
			Result:    StatusUpdateFailed,
			Message:   wrappedErr,
		}
	}

	// Always run update. Should be a no-op if component is already up-to-date.
	err = reconciler.Update(ctx, component)
	if err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusUpdateFailed,
			Message:   err.Error(),
		}
	}

	// Observe Component again after updating it
	observation, err = reconciler.Observe(ctx, component)
	if err != nil {
		return ComponentResult{
			Component: component,
			Result:    StatusObservationFailed,
			Message:   err.Error(),
		}
	}

	// Resource is not healthy
	if !observation.Healthy {
		return ComponentResult{
			Component: component,
			Result:    StatusUnhealthy,
			Message:   observation.Message,
		}
	}

	// Resource is healthy
	return ComponentResult{
		Component: component,
		Result:    StatusHealthy,
		Message:   fmt.Sprintf("%s is healthy.", component.GetName()),
	}
}

func (am *Juggler) findReconcilerFor(component Component) (ComponentReconciler, error) {
	var selectedReconciler ComponentReconciler
	for _, cr := range am.reconcilers {
		for _, kt := range cr.KnownTypes() {
			if reflect.TypeOf(component) == kt {
				if selectedReconciler != nil {
					return nil, errTooManyReconcilers
				}
				selectedReconciler = cr
			}
		}
	}
	if selectedReconciler != nil {
		return selectedReconciler, nil
	}
	return nil, errNoReconcilerForComponentType
}

func (am *Juggler) findRegisteredComponent(sample Component) Component {
	for _, registered := range am.components {
		if reflect.TypeOf(registered) == reflect.TypeOf(sample) {
			return registered
		}
	}
	return nil
}

func (am *Juggler) checkDependencies(component Component) error {
	for _, dep := range component.GetDependencies() {
		registered := am.findRegisteredComponent(dep)
		if registered == nil {
			return errDependencyNotRegistered
		}
		if !registered.IsEnabled() {
			return errDependencyNotEnabled
		}
	}
	return nil
}

func (am *Juggler) componentsOfReconciler(r ComponentReconciler) []Component {
	configuredComponents := []Component{}
	for _, cc := range am.components {
		for _, kt := range r.KnownTypes() {
			if reflect.TypeOf(cc) == kt {
				configuredComponents = append(configuredComponents, cc)
			}
		}
	}
	return configuredComponents
}
