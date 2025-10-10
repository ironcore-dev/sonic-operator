// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	client "github.com/ironcore-dev/switch-operator/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func GetDeviceInfo(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "device-info",
		Short:   "Get device information",
		Example: "switch-proxy-client get device-info",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGetDeviceInfo(cmd.Context(), GetSharedSwitchAgentClient(), printer)
		},
	}

	return cmd
}

func RunGetDeviceInfo(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
) error {

	device, err := c.GetDeviceInfo(ctx)

	if err != nil {
		return fmt.Errorf("failed to get device info: %v", err)
	}

	return printer.Print("Device Info", os.Stdout, device)
}
