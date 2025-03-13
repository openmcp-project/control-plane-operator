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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CrossplanePackageRestrictionSpec defines the desired state of CrossplanePackageRestriction
type CrossplanePackageRestrictionSpec struct {
	Providers      PackageRestriction `json:"providers"`
	Configurations PackageRestriction `json:"configurations"`
	Functions      PackageRestriction `json:"functions"`
}

// PackageRestriction restricts a package type (e.g. providers) to certain registries or literal packages.
// If both Registries and Packages are empty, no packages of this type will be allowed.
type PackageRestriction struct {
	Registries []string `json:"registries"`
	Packages   []string `json:"packages"`
}

// CrossplanePackageRestrictionStatus defines the observed state of CrossplanePackageRestriction
type CrossplanePackageRestrictionStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// CrossplanePackageRestriction is the Schema for the crossplanepackagerestrictions API
type CrossplanePackageRestriction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CrossplanePackageRestrictionSpec   `json:"spec,omitempty"`
	Status CrossplanePackageRestrictionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CrossplanePackageRestrictionList contains a list of CrossplanePackageRestriction
type CrossplanePackageRestrictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CrossplanePackageRestriction `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CrossplanePackageRestriction{}, &CrossplanePackageRestrictionList{})
}
