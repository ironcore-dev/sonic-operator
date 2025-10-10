// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"

	client "github.com/ironcore-dev/switch-operator/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func SubcommandRequired(cmd *cobra.Command, args []string) error {
	if err := cmd.Help(); err != nil {
		return err
	}
	return errors.New("subcommand is required")
}

var switchAgentClient client.SwitchAgentClient

func GetSharedSwitchAgentClient() client.SwitchAgentClient {
	return switchAgentClient
}

func Command() *cobra.Command {
	cobra.EnableCaseInsensitive = true

	cmd := &cobra.Command{
		Use:           "switch-proxy-client [command]",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          SubcommandRequired,
	}

	cmd.AddCommand(
		Get(),
		List(),
		Set(),
		// Create(),
		// Delete(),
	)

	switchAgentClient, _ = client.NewDefaultSwitchAgentClient()

	switchAgentClient.AddSwitchAgentClientFlags(cmd.PersistentFlags())

	return cmd
}
