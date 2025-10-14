// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	server "github.com/ironcore-dev/switch-operator/internal/agent/agent_server"
)

func main() {
	server.StartServer()
}
