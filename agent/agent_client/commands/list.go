// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	client "github.com/ironcore-dev/switch-operator/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func List() *cobra.Command {

	printRenderer := client.NewDefaultPrintRender("table")

	cmd := &cobra.Command{
		Use:  "list [subcommand]",
		Args: cobra.NoArgs,
		RunE: SubcommandRequired,
	}

	subcommands := []*cobra.Command{
		ListInterfaces(printRenderer),
		ListPorts(printRenderer),
	}

	cmd.AddCommand(subcommands...)
	return cmd
}
