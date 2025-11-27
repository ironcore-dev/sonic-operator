// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"

	pb "github.com/ironcore-dev/switch-operator/internal/agent/proto"

	api "github.com/ironcore-dev/switch-operator/api/v1alpha1"
)

func ProtoStatusToStatus(pbStatus *pb.Status) Status {
	if pbStatus == nil {
		return Status{
			Code:    0,
			Message: "",
		}
	}
	return Status{
		Code:    pbStatus.Code,
		Message: pbStatus.Message,
	}
}

func ValidateDeviceStatusStr(status string) (string, error) {
	if status != "up" && status != "down" {
		return "", fmt.Errorf("invalid device status: %s, it has to be 'up' or 'down'", status)
	}
	return status, nil
}

func AgentDeviceStatusToAPIAdminState(deviceStatus DeviceStatus) (api.AdminState, error) {
	switch deviceStatus {
	case StatusUp:
		return api.AdminStateUp, nil
	case StatusDown:
		return api.AdminStateDown, nil
	default:
		return api.AdminStateUnknown, fmt.Errorf("unknown device status: %s", deviceStatus)
	}
}

func APIAdminStateToAgentDeviceStatus(adminState api.AdminState) (DeviceStatus, error) {
	switch adminState {
	case api.AdminStateUp:
		return StatusUp, nil
	case api.AdminStateDown:
		return StatusDown, nil
	default:
		return StatusUnknown, fmt.Errorf("unknown admin state: %s", adminState)
	}
}

func AgentDeviceStatusToAPIOperationState(deviceStatus DeviceStatus) (api.OperationState, error) {
	switch deviceStatus {
	case StatusUp:
		return api.OperationStateUp, nil
	case StatusDown:
		return api.OperationStateDown, nil
	default:
		return api.OperationStateUnknown, fmt.Errorf("unknown device status: %s", deviceStatus)
	}
}

func APIOperationStateToAgentDeviceStatus(opState api.OperationState) (DeviceStatus, error) {
	switch opState {
	case api.OperationStateUp:
		return StatusUp, nil
	case api.OperationStateDown:
		return StatusDown, nil
	default:
		return StatusUnknown, fmt.Errorf("unknown operation state: %s", opState)
	}
}
