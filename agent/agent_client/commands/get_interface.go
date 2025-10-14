// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	client "github.com/ironcore-dev/switch-operator/agent/agent_client/client"
	agent "github.com/ironcore-dev/switch-operator/agent/types"

	"github.com/spf13/cobra"
)

func GetInterface(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "interface",
		Short:   "Get interface information",
		Example: "agent_cli get interface <interface-name>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGetInterface(cmd.Context(), GetSharedSwitchAgentClient(), printer, args[0])
		},
	}

	return cmd
}

func RunGetInterface(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
	interfaceName string,
) error {

	iface, err := c.GetInterface(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: interfaceName,
	})

	if err != nil {
		return fmt.Errorf("failed to get interface info: %v", err)
	}

	return printer.Print("Interface Info", os.Stdout, iface)
}
