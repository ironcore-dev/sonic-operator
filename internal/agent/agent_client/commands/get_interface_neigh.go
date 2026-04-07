// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	var abstractName string

	if strings.HasPrefix(interfaceName, "Ethernet") {
		var err error
		abstractName, err = agent.NativeNameToAbstractName(interfaceName)
		if err != nil {
			return fmt.Errorf("failed to convert native name to abstract name: %v", err)
		}

	} else if strings.HasPrefix(interfaceName, "eth") {
		abstractName = interfaceName
	} else {
		return fmt.Errorf("invalid interface name: %s. Must start with 'Ethernet' or 'eth'", interfaceName)
	}

	ifaceNeigh, err := c.GetInterfaceNeighbor(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name: abstractName,
	})

	if err != nil {
		return fmt.Errorf("failed to get interface neighbor info: %v", err)
	}

	return printer.Print("Interface Neighbor Info", os.Stdout, ifaceNeigh)
}
