/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"github.com/openmcp-project/controller-utils/pkg/api"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LabelControlPlane indicates to which ControlPlane a resource belongs.
	LabelControlPlane = "core.orchestrate.cloud.sap/control-plane"

	// Finalizer is the default finalizer which is added to resources managed by the control-plane-operator.
	Finalizer = "core.orchestrate.cloud.sap"
)

// ControlPlaneSpec defines the desired state of ControlPlane
type ControlPlaneSpec struct {
	// Reference to a core configuration
	// +kubebuilder:default:={name:"default"}
	CoreReference v1.LocalObjectReference `json:"coreRef,omitempty"`

	// Configuration of the ControlPlane target (local or remote cluster)
	Target Target `json:"target"`

	// Configuration for the telemetry.
	// +kubebuilder:validation:Optional
	Telemetry *TelemetryConfig `json:"telemetry,omitempty"`

	// Pull secrets which will be used when pulling charts, providers, etc.
	// +kubebuilder:validation:Optional
	PullSecrets []v1.LocalObjectReference `json:"pullSecrets,omitempty"`

	ComponentsConfig `json:",inline"`
}

type Target struct {
	api.Target `json:",inline"`

	// FluxServiceAccount is a reference to a service account that should be used by Flux.
	// +kubebuilder:validation:Required
	FluxServiceAccount ServiceAccountReference `json:"fluxServiceAccount"`
}

type ServiceAccountReference struct {
	// Name is the name of the service account.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Namespace is the namespace of the service account.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// Overrides specifies fields that should be overwritten when a kubeconfig is generated from this ServiceAccountReference.
	Overrides KubeconfigOverrides `json:"overrides,omitempty"`
}

type KubeconfigOverrides struct {
	// Host must be a host string, a host:port pair, or a URL to the base of the apiserver.
	Host string `json:"host,omitempty"`
}

// TelemetryConfig allows the toggling of telemetry data
type TelemetryConfig struct {
	// Enables or disables telemetry.
	Enabled bool `json:"enabled,omitempty"`
}

// ChartSpec identifies a Helm chart.
type ChartSpec struct {
	// Repository is the URL to a Helm repository
	Repository string `json:"repository,omitempty"`

	// Name of the Helm chart
	Name string `json:"name,omitempty"`

	// Version of the Helm chart, latest version if not set
	Version string `json:"version,omitempty"`
}

// ControlPlaneStatus defines the observed state of ControlPlane
type ControlPlaneStatus struct {
	// Current service state of the ControlPlane.
	Conditions []metav1.Condition `json:"conditions"`

	// Namespace that contains resources related to the ControlPlane.
	Namespace string `json:"namespace"`

	// Number of enabled components.
	ComponentsEnabled int `json:"componentsEnabled"`

	// Number of healthy components.
	ComponentsHealthy int `json:"componentsHealthy"`
}

// ControlPlane is the Schema for the ControlPlane API
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=cp,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Components Healthy",type="integer",JSONPath=".status.componentsHealthy"
// +kubebuilder:printcolumn:name="Components Enabled",type="integer",JSONPath=".status.componentsEnabled"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type ControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneSpec   `json:"spec,omitempty"`
	Status ControlPlaneStatus `json:"status,omitempty"`
}

func (cp ControlPlane) WasDeleted() bool {
	return !cp.DeletionTimestamp.IsZero()
}

//+kubebuilder:object:root=true

// ControlPlaneList contains a list of ControlPlane
type ControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ControlPlane{}, &ControlPlaneList{})
}

// Condition types.
const (
	// TypeReady resources are believed to be ready to handle work.
	TypeReady string = "Ready"

	// TypeSynced resources are believed to be in sync with the
	// Kubernetes resources that manage their lifecycle.
	TypeSynced      string = "Synced"
	TypeReconciling string = "Reconciling"
)

// Reasons a resource is or is not ready.
const (
	ReasonAvailable   string = "Available"
	ReasonUnavailable string = "Unavailable"
	ReasonCreating    string = "Creating"
	ReasonDeleting    string = "Deleting"
)

// Reasons a resource is or is not synced.
const (
	ReasonReconcileSuccess string = "ReconcileSuccess"
	ReasonReconcileError   string = "ReconcileError"
	ReasonReconcilePaused  string = "ReconcilePaused"
)

// Creating returns a condition that indicates the resource is currently
// being created.
func Creating() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonCreating,
	}
}

// Deleting returns a condition that indicates the resource is currently
// being deleted.
func Deleting() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonDeleting,
	}
}

// Unavailable returns a condition that indicates the resource is
// currently observed NOT to be available for use.
func Unavailable() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonUnavailable,
	}
}

// Available returns a condition that indicates the resource is
// currently observed to be available for use.
func Available() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonAvailable,
	}
}

// ReconcileSuccess returns a condition indicating that reconciler successfully
// completed the most recent reconciliation of the resource.
func ReconcileSuccess() metav1.Condition {
	return metav1.Condition{
		Type:               TypeSynced,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconcileSuccess,
	}
}

// ReconcileError returns a condition indicating that controlplane reconciler encountered an
// error while reconciling the resource. This could mean controlplane reconciler was
// unable to update the resource to reflect its desired state, or that
// it was unable to determine the current actual state of the resource.
func ReconcileError(err error) metav1.Condition {
	return metav1.Condition{
		Type:               TypeSynced,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonReconcileError,
		Message:            err.Error(),
	}
}
