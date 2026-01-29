// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"os"
	"time"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func SubcommandRequired(cmd *cobra.Command, args []string) error {
	if err := cmd.Help(); err != nil {
		return err
	}
	return errors.New("subcommand is required")
}

var switchAgentClient client.SwitchAgentClient
var address string
var connectTimeout time.Duration

func GetSharedSwitchAgentClient() client.SwitchAgentClient {
	return switchAgentClient
}

func Command() *cobra.Command {
	cobra.EnableCaseInsensitive = true

	cmd := &cobra.Command{
		Use:           "agent_cli [command]",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          SubcommandRequired,
	}

	cmd.AddCommand(
		Get(),
		List(),
		Set(),
	)

	grpcPort := os.Getenv("SWITCH_PROXY_GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	cmd.PersistentFlags().StringVar(&address, "address", "localhost:"+grpcPort, "switch proxy address (overrides SWITCH_PROXY_GRPC_PORT).")
	cmd.PersistentFlags().DurationVar(&connectTimeout, "connect-timeout", 4*time.Second, "Timeout to connect to the switch proxy.")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		switchAgentClient, err = client.NewDefaultSwitchAgentClient(address, connectTimeout)
		if err != nil {
			return err
		}
		return nil
	}
	return cmd
}
