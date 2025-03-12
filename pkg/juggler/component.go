package juggler

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Component is an interface for manageable components.
type Component interface {
	// GetName returns the name of the component.
	GetName() string

	// GetDependencies returns all dependencies of the component.
	GetDependencies() []Component

	// IsEnabled returns if a component is enabled.
	IsEnabled() bool

	// Hooks returns the hooks for a component.
	Hooks() ComponentHooks

	// IsInstallable returns if a component is installable.
	// And if not, the error which describes why it is not installable.
	IsInstallable(ctx context.Context) (bool, error)
}

// StatusVisibility interface defines methods that can be optionally
// implemented by components. A component can implement this interface
// to indicate that the component is to be considered internal. If the
// component does not implement this interface, it is considered as
// external.
type StatusVisibility interface {
	// IsStatusInternal method implemented by a component
	// indicates that the component is internal or external.
	IsStatusInternal() bool
}

// isComponentInternal function checks whether a component is internal
// or not. A component is internal only if it implements the
// StatusVisibility interface and its IsStatusInternal method returns
// true.
func isComponentInternal(component Component) bool {
	c, ok := component.(StatusVisibility)
	if ok {
		return c.IsStatusInternal()
	}
	return false
}

// KeepOnUninstall can be implemented by components that should not be uninstalled, e.g. CRDs.
type KeepOnUninstall interface {
	KeepOnUninstall() bool
}

// ComponentStatus indicates the status of a component.
type ComponentStatus struct {
	Name       string
	IsReady    bool
	EmitsEvent ComponentEventType
}

//nolint:lll
var (
	// StatusReconcilerNotFound states that no registered reconciler was able to handle the component.
	StatusReconcilerNotFound = ComponentStatus{Name: "ReconcilerNotFound", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusObservationFailed states that the current state (installed, healthy, etc.)
	// of the component could not be determined.
	StatusObservationFailed = ComponentStatus{Name: "ObservationFailed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusUninstallFailed states that a component could not be uninstalled.
	StatusUninstallFailed = ComponentStatus{Name: "UninstallFailed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusUninstalled states that a component has been uninstalled.
	StatusUninstalled = ComponentStatus{Name: "Uninstalled", IsReady: false, EmitsEvent: ComponentEventNormal}

	// StatusDisabled states that a component is disabled and no action has been taken.
	StatusDisabled = ComponentStatus{Name: "Disabled", IsReady: false, EmitsEvent: ComponentEventNone}

	// StatusComponentNotAllowed states that a component is not allowed to be installed.
	StatusComponentNotAllowed = ComponentStatus{Name: "ComponentNotAllowed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusDependencyCheckFailed states that a dependency check failed (e.g. dependency not enabled)
	StatusDependencyCheckFailed = ComponentStatus{Name: "DependencyCheckFailed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusInstallFailed states that a component could not be installed.
	StatusInstallFailed = ComponentStatus{Name: "InstallFailed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusInstalled states that a component has been installed.
	StatusInstalled = ComponentStatus{Name: "Installed", IsReady: false, EmitsEvent: ComponentEventNormal}

	// StatusUpdateFailed states that a component could not be updated.
	StatusUpdateFailed = ComponentStatus{Name: "UpdateFailed", IsReady: false, EmitsEvent: ComponentEventWarning}

	// StatusUnhealthy states that a component is unhealthy.
	StatusUnhealthy = ComponentStatus{Name: "Unhealthy", IsReady: false, EmitsEvent: ComponentEventNone}

	// StatusHealthy states that a component is healthy and no action has been taken.
	StatusHealthy = ComponentStatus{Name: "Healthy", IsReady: true, EmitsEvent: ComponentEventNone}
)

// ComponentResult contains information about an operation performed by the component manager.
type ComponentResult struct {
	Component Component
	Result    ComponentStatus
	Message   string
}

// ComponentHooks defines hooks for a Component.
type ComponentHooks struct {
	PreUninstall func(ctx context.Context, c client.Client) error
	PreInstall   func(ctx context.Context, c client.Client) error
	PreUpdate    func(ctx context.Context, c client.Client) error
}

// ToCondition converts a ComponentResult to a Kubernetes Condition.
func (r ComponentResult) ToCondition() metav1.Condition {
	message := r.Message
	status := metav1.ConditionFalse
	if r.Result.IsReady {
		status = metav1.ConditionTrue
	}
	reason := r.Result.Name
	if reason == "" {
		reason = "Unknown"
		status = metav1.ConditionFalse
	}

	return metav1.Condition{
		Type:               r.conditionType(),
		Status:             status,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
	}
}

// conditionType is a helper method. It calculates the Condition.Type
// field. For internal components, the value starts with a lowercase
// letter.
func (r ComponentResult) conditionType() string {
	componentName := r.Component.GetName()
	if isComponentInternal(r.Component) && len(componentName) > 0 {
		componentName = strings.ToLower(componentName[0:1]) + componentName[1:]
	}
	return fmt.Sprintf("%sReady", componentName)
}
