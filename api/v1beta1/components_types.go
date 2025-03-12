package v1beta1

// ComponentsConfig defines all the different Components that can be installed in a ControlPlane.
type ComponentsConfig struct {
	// Configuration for the Crossplane installation of this ControlPlane.
	// +kubebuilder:validation:Optional
	Crossplane *CrossplaneConfig `json:"crossplane,omitempty"`

	// Configuration for the BTP Service Operator. More info:
	// https://github.com/SAP/sap-btp-service-operator
	// +kubebuilder:validation:Optional
	BTPServiceOperator *BTPServiceOperatorConfig `json:"btpServiceOperator,omitempty"`

	// CertManager configures the cert-manager component. More info:
	// https://cert-manager.io/
	// +kubebuilder:validation:Optional
	CertManager *CertManagerConfig `json:"certManager,omitempty"`

	// Configuration for the External Secrets Operator. More info:
	// https://external-secrets.io
	// +kubebuilder:validation:Optional
	ExternalSecretsOperator *ExternalSecretsOperatorConfig `json:"externalSecretsOperator,omitempty"`

	// Configuration for Kyverno. More info:
	// https://kyverno.io/
	// +kubebuilder:validation:Optional
	Kyverno *KyvernoConfig `json:"kyverno,omitempty"`

	// Configuration for Flux. More info:
	// https://fluxcd.io/
	// +kubebuilder:validation:Optional
	Flux *FluxConfig `json:"flux,omitempty"`
}
