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
	var iface *agent.Interface
	var err error
	// Determine if the provided name is a native name (e.g., "Ethernet0") or an abstract name (eth0-0")
	if strings.HasPrefix(interfaceName, "Ethernet") {
		var abstractName string
		abstractName, err = agent.NativeNameToAbstractName(interfaceName)
		if err != nil {
			return fmt.Errorf("failed to convert native name to abstract name: %v", err)
		}
		iface, err = c.GetInterfaceByAbstractName(ctx, &agent.Interface{
			TypeMeta: agent.TypeMeta{
				Kind: agent.InterfaceKind,
			},
			Name: abstractName,
		})
	} else if strings.HasPrefix(interfaceName, "eth") {
		iface, err = c.GetInterfaceByAbstractName(ctx, &agent.Interface{
			TypeMeta: agent.TypeMeta{
				Kind: agent.InterfaceKind,
			},
			Name: interfaceName,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to get interface info: %v", err)
	}

	return printer.Print("Interface Info", os.Stdout, iface)
}
