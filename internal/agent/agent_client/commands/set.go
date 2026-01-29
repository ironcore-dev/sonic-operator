// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"

	"github.com/spf13/cobra"
)

func Set() *cobra.Command {

	printRenderer := client.NewDefaultPrintRender("table")

	cmd := &cobra.Command{
		Use:  "set [subcommand]",
		Args: cobra.NoArgs,
		RunE: SubcommandRequired,
	}

	subcommands := []*cobra.Command{
		SetInterfaceStatus(printRenderer),
	}

	cmd.AddCommand(subcommands...)
	return cmd
}
