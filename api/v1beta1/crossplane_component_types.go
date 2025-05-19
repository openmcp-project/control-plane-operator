package v1beta1

import (
	crossplanev1 "github.com/crossplane/crossplane/apis/pkg/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// CrossplaneConfig configures the Crossplane component.
type CrossplaneConfig struct {
	// The Version of Crossplane to install.
	Version string `json:"version"`

	// Optional custom Helm chart configuration.
	Chart *ChartSpec `json:"chart,omitempty"`

	// Optional additional values that should be passed to the Crossplane Helm chart.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values *apiextensionsv1.JSON `json:"values,omitempty"`

	// List of Crossplane providers to be installed.
	// +kubebuilder:validation:Optional
	Providers []*CrossplaneProviderConfig `json:"providers,omitempty"`
}

// CrossplaneProviderConfig represents configuration for Crossplane providers in a ControlPlane.
// Primarily based on the Crossplane open source API.
type CrossplaneProviderConfig struct {
	// Name of the provider.
	// Using a well-known name will automatically configure the "package" field.
	Name string `json:"name"`

	// Version of the provider to install.
	Version string `json:"version"`

	// Provider package to be installed.
	// If "name" is set to a well-known value, this field will be configured automatically.
	// +kubebuilder:validation:Optional
	Package string `json:"package,omitempty"`

	// Pull policy for the provider.
	// One of Always, Never, IfNotPresent.
	// +kubebuilder:default=IfNotPresent
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	PackagePullPolicy *corev1.PullPolicy `json:"packagePullPolicy,omitempty"`

	// PackagePullSecrets are named secrets in the same namespace that can be used to fetch packages from private registries.
	PackagePullSecrets []corev1.LocalObjectReference `json:"packagePullSecrets,omitempty"`

	crossplanev1.PackageRuntimeSpec `json:",inline"`
}
