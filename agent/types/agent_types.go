// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"
	"reflect"
)

type Status struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

func (status *Status) String() string {
	if status.Code == 0 {
		return status.Message
	}
	return fmt.Sprintf("Code: %d, Message: %s", status.Code, status.Message)
}

func NewErrorStatus(code uint32, message string) *Status {
	return &Status{
		Code:    code,
		Message: message,
	}
}

func NewCorrectStatus() *Status {
	return &Status{
		Code:    0,
		Message: "",
	}
}

type TypeMeta struct {
	Kind string `json:"kind"`
}

func (m *TypeMeta) GetKind() string {
	return m.Kind
}

type Object interface {
	GetKind() string
	GetName() string
	GetStatus() Status
}

type List interface {
	GetItems() []Object
	GetStatus() Status
}

type DeviceMeta struct {
	LocalMacAddress string `json:"local_mac_address"`
	Hwsku           string `json:"hwsku"`
	SonicOSVersion  string `json:"sonic_os_version"`
	AsicType        string `json:"asic_type"`
}

const (
	StatusDown = iota
	StatusUp
	StatusUnknown
)

const (
	StatusNotReady = iota
	StatusReady
)

type DeviceSpec struct {
	Readiness uint32 `json:"readiness"`
}

type SwitchDevice struct {
	TypeMeta `json:",inline"`

	LocalMacAddress string `json:"local_mac_address"`
	Hwsku           string `json:"hwsku"`
	SonicOSVersion  string `json:"sonic_os_version"`
	AsicType        string `json:"asic_type"`
	Readiness       uint32 `json:"readiness"`

	Status Status `json:"status"`
}

func (d *SwitchDevice) GetName() string {
	return fmt.Sprintf("%s-%s", "switch", d.LocalMacAddress)
}

func (d *SwitchDevice) GetStatus() Status {
	return d.Status
}

type Interface struct {
	TypeMeta `json:",inline"`

	Name            string `json:"name"`
	MacAddress      string `json:"mac_address"`
	OperationStatus uint32 `json:"operation_status"`
	AdminStatus     uint32 `json:"admin_status"`

	Status Status `json:"status"`
}

func (i *Interface) GetName() string {
	return i.Name
}

func (i *Interface) GetStatus() Status {
	return i.Status
}

type InterfaceList struct {
	TypeMeta `json:",inline"`
	Items    []Interface `json:"items"`
	Status   Status      `json:"status"`
}

func (l *InterfaceList) GetItems() []Object {
	items := make([]Object, len(l.Items))
	for i, item := range l.Items {
		items[i] = &item
	}
	return items
}

func (l *InterfaceList) GetStatus() Status {
	return l.Status
}

type InterfaceNeighbor struct {
	TypeMeta `json:",inline"`
	Name     string `json:"name"` // Interface name of yourself

	MacAddress string `json:"mac_address"`
	SystemName string `json:"system_name"`
	Handle     string `json:"handle"`

	Status Status `json:"status"`
}

func (n *InterfaceNeighbor) GetName() string {
	return n.Name
}

func (n *InterfaceNeighbor) GetStatus() Status {
	return n.Status
}

type InterfaceNeighborList struct {
	TypeMeta `json:",inline"`
	Items    []InterfaceNeighbor `json:"items"`
	Status   Status              `json:"status"`
}

func (l *InterfaceNeighborList) GetItems() []Object {
	items := make([]Object, len(l.Items))
	for i, item := range l.Items {
		items[i] = &item
	}
	return items
}

func (l *InterfaceNeighborList) GetStatus() Status {
	return l.Status
}

type Port struct {
	TypeMeta `json:",inline"`
	Name     string `json:"name"`

	Alias  string `json:"alias"`
	Status Status `json:"status"`
}

func (p *Port) GetName() string {
	return p.Name
}

func (p *Port) GetStatus() Status {
	return p.Status
}

type PortList struct {
	TypeMeta `json:",inline"`
	Items    []Port `json:"items"`
	Status   Status `json:"status"`
}

func (l *PortList) GetItems() []Object {
	items := make([]Object, len(l.Items))
	for i, item := range l.Items {
		items[i] = &item
	}
	return items
}

func (l *PortList) GetStatus() Status {
	return l.Status
}

var (
	DeviceKind                = reflect.TypeOf(SwitchDevice{}).Name()
	InterfaceKind             = reflect.TypeOf(Interface{}).Name()
	InterfaceListKind         = reflect.TypeOf(InterfaceList{}).Name()
	PortKind                  = reflect.TypeOf(Port{}).Name()
	PortListKind              = reflect.TypeOf(PortList{}).Name()
	InterfaceNeighborKind     = reflect.TypeOf(InterfaceNeighbor{}).Name()
	InterfaceNeighborListKind = reflect.TypeOf(InterfaceNeighborList{}).Name()
)
