package v1beta1

import apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

// BTPServiceOperatorConfig configures the BTP Service Operator component.
type BTPServiceOperatorConfig struct {
	// The Version of BTP Service Operator to install.
	Version string `json:"version"`

	// Optional custom chart configuration.
	Chart *ChartSpec `json:"chart,omitempty"`

	// Optional additional values that should be passed to the BTP Service Operator Helm chart.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}
