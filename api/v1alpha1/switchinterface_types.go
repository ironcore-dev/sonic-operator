// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdminState string

const (
	AdminStateUp   AdminState = "Up"
	AdminStateDown AdminState = "Down"
)

// SwitchInterfaceSpec defines the desired state of SwitchInterface
type SwitchInterfaceSpec struct {
	// SwitchRef is a reference to the Switch this interface is connected to.
	// +required
	SwitchRef v1.LocalObjectReference `json:"switchRef"`

	// AdminState represents the desired administrative state of the interface.
	// +optional
	AdminState AdminState `json:"adminState,omitempty"`
}

type OperationState string

const (
	OperationStateUp   OperationState = "Up"
	OperationStateDown OperationState = "Down"
)

type SwitchInterfaceState string

const (
	SwitchInterfaceStatePending SwitchInterfaceState = "Pending"
	SwitchInterfaceStateReady   SwitchInterfaceState = "Ready"
	SwitchInterfaceStateFailed  SwitchInterfaceState = "Failed"
)

// Neighbor represents a connected neighbor device.
type Neighbor struct {
	// MacAddress is the MAC address of the neighbor device.
	MacAddress string `json:"macAddress,omitempty"`

	// SystemName is the name of the neighbor device.
	SystemName string `json:"systemName,omitempty"`
}

// SwitchInterfaceStatus defines the observed state of SwitchInterface.
type SwitchInterfaceStatus struct {
	// AdminState represents the desired administrative state of the interface.
	// +optional
	AdminState AdminState `json:"adminState,omitempty"`

	// OperationalState represents the actual operational state of the interface.
	// +optional
	OperationalState OperationState `json:"operationalState,omitempty"`

	// State represents the high-level state of the SwitchInterface.
	// +optional
	State SwitchInterfaceState `json:"state,omitempty"`

	// Neighbor is a reference to the connected neighbor device, if any.
	// +optional
	Neighbor Neighbor `json:"neighbor,omitempty"`

	// MacAddress is the MAC address assigned to this interface.
	MacAddress string `json:"macAddress,omitempty"`

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

// SwitchInterface is the Schema for the switchinterfaces API
type SwitchInterface struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of SwitchInterface
	// +required
	Spec SwitchInterfaceSpec `json:"spec"`

	// status defines the observed state of SwitchInterface
	// +optional
	Status SwitchInterfaceStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// SwitchInterfaceList contains a list of SwitchInterface
type SwitchInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwitchInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwitchInterface{}, &SwitchInterfaceList{})
}
