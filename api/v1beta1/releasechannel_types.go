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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=false
type VersionResolverFn func(componentName string, version string) (ComponentVersion, error)

// ReleaseChannelSpec defines the desired state of ReleaseChannel
type ReleaseChannelSpec struct {
	// Specify a ocm registry url where the releasechannel components are uploaded
	// +kubebuilder:validation:MinLength=1
	OcmRegistryUrl string `json:"ocmRegistryUrl,omitempty"`
	// This should be a reference to a secret, which has the `username` and `password` keys.
	// If specified, will be used when accessing the ocmRegistry specified in ocmRegistryUrl.
	PullSecretRef corev1.SecretReference `json:"pullSecretRef,omitempty"`

	// This parameter can be used for a tar based ocm registry in a secret.
	// The secret referenced here must contain a key where a tar based ocm registry is stored in.
	OcmRegistrySecretRef corev1.SecretReference `json:"ocmRegistrySecretRef,omitempty"`
	// Here you must specify the key which contains the tar based ocm registry in the referenced secret.
	// Required, if ocmRegistrySecretRef is specified.
	OcmRegistrySecretKey string `json:"ocmRegistrySecretKey,omitempty"`

	// When specified only components starting with this prefix will be fetched.
	// Also this prefix will be cut from the componentNames in the status field.
	PrefixFilter string `json:"prefixFilter,omitempty"`

	// Interval specifies the timespan when the registry is checked again
	// +kubebuilder:default="15m"
	Interval metav1.Duration `json:"interval"`
}

// ReleaseChannelStatus defines the observed state of ReleaseChannel
type ReleaseChannelStatus struct {
	// The components which are inside the ocm registry
	Components []Component `json:"components,omitempty"`
}

type Component struct {
	// Name of the component which can be used to install it via the controlplane CR.
	Name string `json:"name"`
	// All available versions for that component.
	Versions []ComponentVersion `json:"versions"`
}

type ComponentVersion struct {
	// The version number for that ComponentVersion
	Version string `json:"version"`
	// if it's a Docker Image, this specifies the Docker reference for pulling the image
	DockerRef string `json:"dockerRef,omitempty"`
	// if it's a helm chart, this specifies the helm repo
	HelmRepo string `json:"helmRepo,omitempty"`
	// if it's a helm chart, this specifies the chart name
	HelmChart string `json:"helmChart,omitempty"`
	// if the Helm chart is stored in an OCI registry, this specifies the OCI URL
	OCIURL string `json:"ociUrl,omitempty"`
}

// ReleaseChannel is the Schema for the ReleaseChannel API
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=rc,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:validation:XValidation:rule=(!(has(self.spec.ocmRegistryUrl) && has(self.spec.ocmRegistrySecretRef))), message="You can't specify 'ocmRegistryUrl' and 'ocmRegistrySecretRef' at the same time, either use a remote ocm registry or a secret"
// +kubebuilder:validation:XValidation:rule=(!(has(self.spec.ocmRegistrySecretRef) && !has(self.spec.ocmRegistrySecretKey))), message="You need to specify an 'ocmRegistrySecretKey' if you want to use the 'ocmRegistrySecretRef'."
// +kubebuilder:validation:XValidation:rule=(!(has(self.spec.pullSecretRef) && !has(self.spec.ocmRegistryUrl))), message="If you specify a 'pullSecretRef' you must specify an 'ocmRegistryUrl' otherwise the 'pullSecretRef' will not be used."
type ReleaseChannel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseChannelSpec   `json:"spec,omitempty"`
	Status ReleaseChannelStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ReleaseChannelList contains a list of ReleaseChannel
type ReleaseChannelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReleaseChannel `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReleaseChannel{}, &ReleaseChannelList{})
}
