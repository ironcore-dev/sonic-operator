// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
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
		RebootCause(),
		FactoryReset(),
		GetReadiness(),
		Reprovision(),
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

func RebootCause() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reboot-cause",
		Short:   "Get the cause of the last reboot",
		Example: "agent_cli gnoi reboot-cause",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRebootCause(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunRebootCause(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	cause, err := c.RebootCause(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, cause)
	return err
}

func FactoryReset() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "factory-reset",
		Short:   "Reset the switch to factory defaults",
		Example: "agent_cli gnoi factory-reset",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunFactoryReset(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunFactoryReset(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.FactoryReset(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, "Factory reset command sent successfully")
	return err
}

func GetReadiness() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-readiness",
		Short:   "Check whether the device host-services are ready",
		Example: "agent_cli gnoi get-readiness",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGetReadiness(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunGetReadiness(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	ready, err := c.GetReadiness(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(os.Stdout, "ready: %v\n", ready)
	return err
}

func Reprovision() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reprovision",
		Short:   "Reprovision the switch (triggers ONIE install flow)",
		Example: "agent_cli gnoi reprovision",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunReprovision(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunReprovision(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.Reprovision(ctx)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, "Reprovision command sent successfully")
	return err
}
