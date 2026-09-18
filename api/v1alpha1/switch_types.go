// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	SwitchFinalizer = "sonic.networking.metal.ironcore.dev/sonic-operator"
)

type PortSpec struct {
	Name string `json:"name"`
}

type Management struct {
	Host        string             `json:"host"`
	Port        string             `json:"port"`
	Credentials v1.ObjectReference `json:"credentials"`
}

// ZTPConfigMapReference identifies the ConfigMap entry containing a switch's
// complete ZTP script. The referenced ConfigMap may be in a provisioning
// namespace chosen by the cluster administrator.
type ZTPConfigMapReference struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Key       string `json:"key"`
}

// ZTP defines how a switch is identified while it is requesting its initial
// ZTP script. SourceAddress is the address observed by the provisioning server,
// which can differ from the management address used after provisioning.
type ZTP struct {
	SourceAddress string `json:"sourceAddress"`

	// ScriptRef identifies the complete script served when --ztp-mode=configmap.
	// It is not used by --ztp-mode=generated.
	// +optional
	ScriptRef *ZTPConfigMapReference `json:"scriptRef,omitempty"`
}

// Container declares a Docker container that the provisioning server starts on
// the switch during generated ZTP provisioning.
type Container struct {
	// Name is both the Docker container name and its stable identity on the switch.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// Image is the container image pulled by Docker on the switch.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Command overrides the image entrypoint.
	// +optional
	Command []string `json:"command,omitempty"`

	// Args are appended after Command, or after the image entrypoint when Command is empty.
	// +optional
	Args []string `json:"args,omitempty"`

	// VolumeMounts describes the volumes mounted into the container. Each mount
	// name must refer to an entry in SwitchSpec.Volumes.
	// +optional
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`

	// SecurityContext configures the Unix identity used to run the container.
	// +optional
	SecurityContext *ContainerSecurityContext `json:"securityContext,omitempty"`

	// InjectControlKubeconfig mounts the operator's configured control kubeconfig
	// into this container and sets KUBECONFIG to its in-container path. A
	// securityContext with runAsUser is required so the generated script can
	// grant access to a private, per-container credential file.
	// +optional
	InjectControlKubeconfig bool `json:"injectControlKubeconfig,omitempty"`
}

// ContainerSecurityContext is the supported subset of Kubernetes
// container securityContext for generated Docker containers.
type ContainerSecurityContext struct {
	// RunAsUser is the numeric Unix user ID used by the container.
	// +optional
	// +kubebuilder:validation:Minimum=0
	RunAsUser *int64 `json:"runAsUser,omitempty"`

	// RunAsGroup is the numeric Unix group ID used by the container. It requires
	// runAsUser to be set as well.
	// +optional
	// +kubebuilder:validation:Minimum=0
	RunAsGroup *int64 `json:"runAsGroup,omitempty"`
}

// Volume represents a named storage volume made available to Switch containers.
// It follows the Kubernetes Pod volume model. Generated ZTP currently supports
// hostPath volumes only.
type Volume struct {
	// Name is the stable volume identity referenced by Container.VolumeMounts.
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`

	// HostPath represents a pre-existing file or directory on the SONiC host.
	// +optional
	HostPath *HostPathVolumeSource `json:"hostPath,omitempty"`
}

// HostPathVolumeSource represents a host directory or file mounted into a
// Switch container.
type HostPathVolumeSource struct {
	// Path is the absolute path on the SONiC host.
	Path string `json:"path"`

	// Type describes the expected host-path type, following the Kubernetes Pod
	// hostPath API. Generated ZTP does not create missing paths.
	// +optional
	Type *v1.HostPathType `json:"type,omitempty"`
}

// VolumeMount describes a volume mounted into a Switch container. It follows
// the Kubernetes Pod volumeMount API.
type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
}

// NextBootMode describes the desired behavior of the switch's next boot.
// Providers translate this high-level intent to their platform-specific boot
// mechanism. For SONiC, InstallOS enters ONIE install discovery.
type NextBootMode string

const (
	// NextBootModeNone leaves the normal installed network OS boot path intact.
	NextBootModeNone NextBootMode = "None"
	// NextBootModeInstallOS enters the platform's OS installation/discovery flow.
	NextBootModeInstallOS NextBootMode = "InstallOS"
)

// SwitchSpec defines the desired state of Switch
type SwitchSpec struct {
	// Hostname is configured on the switch by --ztp-mode=generated. If omitted,
	// the generated script uses the Switch object name.
	// +optional
	Hostname string `json:"hostname,omitempty"`

	Management Management `json:"management,omitempty"`

	// ZTP identifies the switch while it requests its initial provisioning script.
	// +optional
	ZTP *ZTP `json:"ztp,omitempty"`

	// Containers are started with host networking and Docker's unless-stopped
	// restart policy by --ztp-mode=generated.
	// +optional
	// +listType=map
	// +listMapKey=name
	Containers []Container `json:"containers,omitempty"`

	// Volumes are named storage sources available to containers. Generated ZTP
	// currently supports hostPath volumes only.
	// +optional
	// +listType=map
	// +listMapKey=name
	Volumes []Volume `json:"volumes,omitempty"`

	// NextBootMode declares the desired behavior of the next boot. The default
	// is None. It is configured by --ztp-mode=generated.
	// +optional
	// +kubebuilder:validation:Enum=None;InstallOS
	NextBootMode NextBootMode `json:"nextBootMode,omitempty"`

	// MacAddress is the MAC address assigned to this interface.
	MacAddress string `json:"macAddress"`

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

	// MACAddress is the MAC address assigned to this switch.
	MACAddress string `json:"macAddress,omitempty"`

	// FirmwareVersion is the firmware version running on this switch.
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// SKU is the stock keeping unit of this switch.
	SKU string `json:"sku,omitempty"`

	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="MACAddress",type=string,JSONPath=`.status.macAddress`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:ac:generate=false
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
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(GroupVersion, &Switch{}, &SwitchList{})
		return nil
	})
}
