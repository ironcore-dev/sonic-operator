// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ironcore-dev/sonic-operator/internal/onie"
)

func main() {
	err := Main()
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}

func Main() error {
	if len(os.Args) != 4 {
		return fmt.Errorf("usage: provisioning-server <listen-addr> <onie-images-dir> <onie-config>")
	}

	listenAddr := os.Args[1]
	onieImagesDir := os.Args[2]
	onieConfigPath := os.Args[3]
	of, err := os.Open(onieConfigPath)
	if err != nil {
		return fmt.Errorf("unable to open onie config file: %w", err)
	}

	var onieConf onie.Config
	if err := json.NewDecoder(of).Decode(&onieConf); err != nil {
		return err
	}

	mux := http.NewServeMux()

	onie.Register(mux, onieImagesDir, onieConf)

	return http.ListenAndServe(listenAddr, mux)
}
