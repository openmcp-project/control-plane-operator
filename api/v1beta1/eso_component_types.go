package v1beta1

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// ExternalSecretsOperatorConfig configures the ExternalSecrets Operator component.
type ExternalSecretsOperatorConfig struct {
	// The Version of External Secrets Operator to install.
	Version string `json:"version"`

	// Optional custom chart configuration.
	Chart *ChartSpec `json:"chart,omitempty"`

	// Optional additional values that should be passed to the External Secrets Operator Helm chart.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}
