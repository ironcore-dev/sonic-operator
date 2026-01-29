// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func ListPorts(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ports",
		Short:   "List ports",
		Example: "agent_cli list ports",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListPorts(cmd.Context(), GetSharedSwitchAgentClient(), printer)
		},
	}

	return cmd
}

func RunListPorts(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
) error {
	ports, err := c.ListPorts(ctx)
	if err != nil {
		return fmt.Errorf("failed to list ports: %v", err)
	}

	return printer.Print("Ports", os.Stdout, ports)
}
