// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PortSpec struct {
	Name string `json:"name"`
}

type Management struct {
	Host        string             `json:"host"`
	Port        string             `json:"port"`
	Credentials v1.ObjectReference `json:"credentials"`
}

// SwitchSpec defines the desired state of Switch
type SwitchSpec struct {
	Management Management `json:"management,omitempty"`

	// Ports the physical ports available on the Switch.
	Ports []PortSpec `json:"ports,omitempty"`
}

// SwitchState represents the high-level state of the Switch.
type SwitchState string

const (
	SwitchStatePending SwitchState = "Pending"
	SwitchStateReady   SwitchState = "Ready"
	SwitchStateFailed  SwitchState = "Failed"
)

// PortStatus defines the observed state of a port on the Switch.
type PortStatus struct {
	// Name is the name of the port.
	Name string `json:"name"`
	// InterfaceRefs lists the references to Interfaces connected to this port.
	InterfaceRefs []v1.LocalObjectReference `json:"interfaceRefs,omitempty"`
}

// SwitchStatus defines the observed state of Switch.
type SwitchStatus struct {
	// State represents the high-level state of the Switch.
	// +optional
	State SwitchState `json:"state,omitempty"`

	// Ports represents the status of each port on the Switch.
	// +optional
	Ports []PortStatus `json:"ports,omitempty"`

	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Switch is the Schema for the switch API
type Switch struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Switch
	// +required
	Spec SwitchSpec `json:"spec"`

	// status defines the observed state of Switch
	// +optional
	Status SwitchStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// SwitchList contains a list of Switch
type SwitchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Switch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Switch{}, &SwitchList{})
}
