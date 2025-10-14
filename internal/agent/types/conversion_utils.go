// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"

	pb "github.com/ironcore-dev/switch-operator/internal/agent/proto"
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

func InterfaceStatusStrToNum(status string) (uint32, error) {
	switch status {
	case "up":
		return 1, nil
	case "down":
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown interface status: %s", status)
	}
}
