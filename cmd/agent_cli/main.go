// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	commands "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/commands"
)

func main() {

	// Initialize the command
	cmd := commands.Command()

	// Execute the command
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
