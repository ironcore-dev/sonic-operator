// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"
)

type DeleteSwitchOptions struct {
	Device string
	Wait   bool
}

func DeleteSwitchCmd() *cobra.Command {
	opts := &DeleteSwitchOptions{}

	cmd := &cobra.Command{
		Use:   "delete-switch",
		Short: "Delete switch configuration and trigger reprovisioning",
		Example: `  # Trigger reprovisioning for leaf-1:
  agent_cli set delete-switch --device leaf-1

  # Trigger reprovisioning and wait until it completes:
  agent_cli set delete-switch --device leaf-1 --wait`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDeleteSwitch(cmd.Context(), GetSharedSwitchAgentClient(), *opts)
		},
	}

	cmd.Flags().StringVar(&opts.Device, "device", "", "Device identifier (e.g. leaf-1)")
	cmd.Flags().BoolVar(&opts.Wait, "wait", false, "Wait until reprovisioning completes (state becomes empty)")
	_ = cmd.MarkFlagRequired("device")

	return cmd
}

func RunDeleteSwitch(ctx context.Context, c client.SwitchAgentClient, opts DeleteSwitchOptions) error {
	if opts.Wait {
		if err := c.EnsureReprovision(ctx, opts.Device); err != nil {
			return fmt.Errorf("EnsureReprovision failed: %w", err)
		}
		_, err := fmt.Fprintf(os.Stdout, "DeleteSwitch completed for device %q\n", opts.Device)
		return err
	}

	state, err := c.DeleteSwitch(ctx, opts.Device)
	if err != nil {
		return fmt.Errorf("DeleteSwitch failed: %w", err)
	}

	if state == "" {
		_, err = fmt.Fprintf(os.Stdout, "DeleteSwitch: device %q is ready (no reprovisioning in progress)\n", opts.Device)
	} else {
		_, err = fmt.Fprintf(os.Stdout, "DeleteSwitch triggered for device %q (state: %q)\n", opts.Device, state)
	}
	return err
}
