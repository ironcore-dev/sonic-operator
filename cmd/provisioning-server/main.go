// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ironcore-dev/switch-operator/internal/onie"
	"github.com/ironcore-dev/switch-operator/internal/ztp"
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
		return fmt.Errorf("usage: provisioning-server <listen-addr> <onie-dir> <ztp-conf>")
	}

	listenAddr := os.Args[1]
	onieInstallerDir := os.Args[2]
	ztpConfigPath := os.Args[3]

	f, err := os.Open(ztpConfigPath)
	if err != nil {
		return fmt.Errorf("unable to open ztp config file: %w", err)
	}

	var ztpConf ztp.Config
	err = json.NewDecoder(f).Decode(&ztpConf)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()

	ztp.Register(mux, ztpConf)
	onie.Register(mux, onieInstallerDir)

	return http.ListenAndServe(listenAddr, mux)
}
