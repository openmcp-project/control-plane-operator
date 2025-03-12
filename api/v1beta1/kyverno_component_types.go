package v1beta1

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// KyvernoConfig configures Kyverno component.
type KyvernoConfig struct {
	// The Version of Kyverno to install.
	Version string `json:"version"`

	// Optional custom chart configuration.
	Chart *ChartSpec `json:"chart,omitempty"`

	// Optional additional values that should be passed to the Kyverno Helm chart.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}
