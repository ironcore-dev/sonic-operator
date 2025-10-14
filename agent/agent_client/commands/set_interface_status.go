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

type SetInterfaceStatusOptions struct {
	InterfaceName string
	AdminStatus   string
}

func SetInterfaceStatus(printer client.PrintRenderer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "interface-status",
		Short:   "Set interface status",
		Example: "agent_cli set interface-status <interface-name> up/down",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := SetInterfaceStatusOptions{
				InterfaceName: args[0],
				AdminStatus:   args[1],
			}
			return RunSetInterfaceStatus(cmd.Context(), GetSharedSwitchAgentClient(), printer, opts)
		},
	}
	return cmd
}

func RunSetInterfaceStatus(
	ctx context.Context,
	c client.SwitchAgentClient,
	printer client.PrintRenderer,
	opts SetInterfaceStatusOptions,
) error {
	adminStatus, err := agent.InterfaceStatusStrToNum(opts.AdminStatus)
	if err != nil {
		return err
	}

	fmt.Printf("Setting interface admin status to: %d for interface: %s\n", adminStatus, opts.InterfaceName)
	iface, err := c.SetInterfaceAdminStatus(ctx, &agent.Interface{
		TypeMeta: agent.TypeMeta{
			Kind: agent.InterfaceKind,
		},
		Name:        opts.InterfaceName,
		AdminStatus: adminStatus,
	})
	if err != nil {
		return fmt.Errorf("failed to set interface admin status: %w", err)
	}

	return printer.Print("Interface admin status updated successfully", os.Stdout, iface)
}
