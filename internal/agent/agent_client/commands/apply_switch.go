// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"context"
	"fmt"
	"os"

	pb "github.com/ironcore-dev/sonic-operator/pkg/agent/proto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"
)

type ApplySwitchOptions struct {
	Device     string
	ConfigFile string
}

// ApplySwitch returns a cobra command that reads a SwitchConfig JSON file and
// calls the FabricSonicSwitchService.ApplySwitch RPC.
//
// The config file must be a JSON document matching the proto SwitchConfig message
// (protojson format — camelCase field names, e.g. {"hostname":"leaf-1","vlans":[...]}).
func ApplySwitchCmd() *cobra.Command {
	opts := &ApplySwitchOptions{}

	cmd := &cobra.Command{
		Use:   "apply-switch",
		Short: "Apply a full switch configuration",
		Example: `  # Apply a switch config from a JSON file:
  agent_cli set apply-switch --device leaf-1 --config-file ./switch.json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunApplySwitch(cmd.Context(), GetSharedSwitchAgentClient(), *opts)
		},
	}

	cmd.Flags().StringVar(&opts.Device, "device", "", "Device identifier (e.g. leaf-1)")
	cmd.Flags().StringVar(&opts.ConfigFile, "config-file", "", "Path to a JSON file containing the SwitchConfig (protojson format)")
	_ = cmd.MarkFlagRequired("device")
	_ = cmd.MarkFlagRequired("config-file")

	return cmd
}

func RunApplySwitch(ctx context.Context, c client.SwitchAgentClient, opts ApplySwitchOptions) error {
	data, err := os.ReadFile(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("reading config file %q: %w", opts.ConfigFile, err)
	}

	cfg := &pb.SwitchConfig{}
	if err := protojson.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parsing config file %q as SwitchConfig JSON: %w", opts.ConfigFile, err)
	}

	if err := c.ApplySwitch(ctx, opts.Device, cfg); err != nil {
		return fmt.Errorf("ApplySwitch failed: %w", err)
	}

	_, err = fmt.Fprintf(os.Stdout, "ApplySwitch applied successfully for device %q\n", opts.Device)
	return err
}
