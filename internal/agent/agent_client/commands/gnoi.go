// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"
)

func Gnoi() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "gnoi [subcommand]",
		Args: cobra.NoArgs,
		RunE: SubcommandRequired,
	}

	subcommands := []*cobra.Command{
		SaveConfig(),
		Reboot(),
		OnieBootModeInstall(),
		RestartSystemdService(),
	}

	cmd.AddCommand(subcommands...)
	return cmd
}

func SaveConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "save-config",
		Short:   "Save the current configuration",
		Example: "agent_cli gnoi save-config",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSaveConfig(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunSaveConfig(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.SaveConfig(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, "Config saved successfully")
	return err
}

func Reboot() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reboot",
		Short:   "Reboot the switch",
		Example: "agent_cli gnoi reboot",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunReboot(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunReboot(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.Reboot(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, "Reboot command sent successfully")
	return err
}

func OnieBootModeInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "onie-boot-mode-install",
		Short:   "set next entry to ONIE boot mode",
		Example: "agent_cli gnoi onie-boot-mode-install",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunOnieBootModeInstall(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunOnieBootModeInstall(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.OnieBootModeInstall(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, "ONIE boot mode set successfully")
	return err
}

func RestartSystemdService() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restart-systemd-service [service-name]",
		Short:   "Restart a systemd service",
		Example: "agent_cli gnoi restart-systemd-service <service-name>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			serviceName := args[0]
			return RunRestartSystemdService(cmd.Context(), GetSharedSwitchAgentClient(), serviceName)
		},
	}
	return cmd
}

func RunRestartSystemdService(
	ctx context.Context,
	c client.SwitchAgentClient,
	serviceName string,
) error {
	err := c.RestartSystemdService(ctx, serviceName)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(os.Stdout, "Systemd service %q restarted successfully\n", serviceName)
	return err
}
