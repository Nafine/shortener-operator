/*
Copyright 2026.

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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UrlShortenerSpec defines the desired state of UrlShortener

type Http struct {
	// +optional
	Port *int32 `json:"port" json-default:"8080"`
	// +optional
	Timeout metav1.Duration `json:"timeout" json-default:"10s"`
	// +optional
	IdleTimeout metav1.Duration `json:"idleTimeout" json-default:"10s"`
}

type UrlShortenerSpec struct {
	// http defines basic configuration of shortener service
	// +optional
	Http `json:"http"`
	// storageDsn defines storage uri
	// +required
	StorageDsnSecretRef *v1.SecretKeySelector `json:"storageDsnSecretRef"`
	// apiKeys defines privileged users
	// +required
	ApiKeysSecretRef *v1.SecretKeySelector `json:"apiKeysSecretRef"`
	// appEnv defines current environment (local|prod)
	// +optional
	AppEnv *string `json:"appEnv" json-default:"local"`
}

// UrlShortenerStatus defines the observed state of UrlShortener.
type UrlShortenerStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// UrlShortener is the Schema for the urlshorteners API
type UrlShortener struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of UrlShortener
	// +required
	Spec UrlShortenerSpec `json:"spec"`

	// status defines the observed state of UrlShortener
	// +optional
	Status UrlShortenerStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// UrlShortenerList contains a list of UrlShortener
type UrlShortenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []UrlShortener `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UrlShortener{}, &UrlShortenerList{})
}
