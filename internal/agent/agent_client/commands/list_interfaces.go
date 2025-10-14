// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	client "github.com/ironcore-dev/switch-operator/internal/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func ListInterfaces(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "interfaces",
		Short:   "List interfaces",
		Example: "agent_cli list interfaces",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListInterfaces(cmd.Context(), GetSharedSwitchAgentClient(), printer)
		},
	}

	return cmd
}

func RunListInterfaces(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
) error {
	interfaces, err := c.ListInterfaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to list interfaces: %v", err)
	}

	return printer.Print("Interfaces", os.Stdout, interfaces)
}
