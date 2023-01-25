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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceKindGarbage = "Garbage"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// GarbageSpec defines the desired state of Garbage
type GarbageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Nginx        *Nginx       `json:"nginx,omitempty"`
	SetOwn       bool         `json:"setOwn,omitempty"`
	SetFinalizer SetFinalizer `json:"setFinalizer,omitempty"`
}

// GarbageStatus defines the observed state of Garbage
type GarbageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	AvailableReplicas int32       `json:"availableReplicas"`
	Conditions        []Condition `json:"conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Garbage is the Schema for the garbages API
type Garbage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GarbageSpec   `json:"spec,omitempty"`
	Status GarbageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GarbageList contains a list of Garbage
type GarbageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Garbage `json:"items"`
}

type Nginx struct {
	Replica *int32  `json:"replica,omitempty"`
	Image   *string `json:"image,omitempty"`
}

type SetFinalizer struct {
	Set  bool    `json:"set,omitempty"`
	Name *string `json:"name,omitempty"`
}

type Condition struct {
	// Type of the condition being reported.
	// +required
	Type ConditionType `json:"type"`
	// Status of the condition.
	// +required
	Status ConditionStatus `json:"status"`
	// lastTransitionTime is the time of the last update to the current status property.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// Reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details for the condition's last transition.
	// +optional
	Message string `json:"message,omitempty"`
	// ObservedGeneration represents the .metadata.generation that the
	// condition was set based upon. For instance, if `.metadata.generation` is
	// currently 12, but the `.status.conditions[].observedGeneration` is 9, the
	// condition is out of date with respect to the current state of the
	// instance.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

type ConditionType string

const (
	// Available indicates whether enough pods are ready to provide the
	// service.
	// The possible status values for this condition type are:
	// - True: all pods are running and ready, the service is fully available.
	// - Degraded: some pods aren't ready, the service is partially available.
	// - False: no pods are running, the service is totally unavailable.
	// - Unknown: the operator couldn't determine the condition status.
	Available ConditionType = "Available"
	// Reconciled indicates whether the operator has reconciled the state of
	// the underlying resources with the object's spec.
	// The possible status values for this condition type are:
	// - True: the reconciliation was successful.
	// - False: the reconciliation failed.
	// - Unknown: the operator couldn't determine the condition status.
	Reconciled ConditionType = "Reconciled"
)

type ConditionStatus string

const (
	ConditionTrue     ConditionStatus = "True"
	ConditionDegraded ConditionStatus = "Degraded"
	ConditionFalse    ConditionStatus = "False"
	ConditionUnknown  ConditionStatus = "Unknown"
)

func init() {
	SchemeBuilder.Register(&Garbage{}, &GarbageList{})
}
