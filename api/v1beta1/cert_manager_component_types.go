package v1beta1

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// CertManagerConfig configures the Cert Manager component.
type CertManagerConfig struct {
	// The Version of the cert-manager to install.
	Version string `json:"version"`

	// Optional custom chart configuration.
	Chart *ChartSpec `json:"chart,omitempty"`

	// Optional additional values that should be passed to the cert-manager Helm chart.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}
