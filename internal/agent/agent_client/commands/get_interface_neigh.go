// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"
	agent "github.com/ironcore-dev/sonic-operator/internal/agent/types"

	"github.com/spf13/cobra"
)

func GetInterfaceNeighbor(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "interface-neighbor",
		Short:   "Get interface neighbors information",
		Example: "agent_cli get interface-neighbor <interface-name>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGetInterfaceNeighbors(cmd.Context(), GetSharedSwitchAgentClient(), printer, args[0])
		},
	}

	return cmd
}

func RunGetInterfaceNeighbors(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
	interfaceName string,
) error {
	ifaceNeigh, err := c.GetInterfaceNeighbor(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: interfaceName,
	})

	if err != nil {
		return fmt.Errorf("failed to get interface neighbor info: %v", err)
	}

	return printer.Print("Interface Neighbor Info", os.Stdout, ifaceNeigh)
}
