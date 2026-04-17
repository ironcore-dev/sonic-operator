// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/ironcore-dev/sonic-operator/internal/onie"
	"github.com/ironcore-dev/sonic-operator/internal/ztp"
)

func main() {
	err := Main()
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}

func Main() error {
	if len(os.Args) != 5 {
		return fmt.Errorf("usage: provisioning-server <listen-addr> <onie-images-dir> <onie-config> <ztp-conf>")
	}

	listenAddr := os.Args[1]
	onieImagesDir := os.Args[2]
	onieConfigPath := os.Args[3]
	ztpConfigPath := os.Args[4]

	f, err := os.Open(ztpConfigPath)
	if err != nil {
		return fmt.Errorf("unable to open ztp config file: %w", err)
	}

	var ztpConf ztp.Config
	err = json.NewDecoder(f).Decode(&ztpConf)
	if err != nil {
		return err
	}

	of, err := os.Open(onieConfigPath)
	if err != nil {
		return fmt.Errorf("unable to open onie config file: %w", err)
	}

	var onieConf onie.Config
	if err := json.NewDecoder(of).Decode(&onieConf); err != nil {
		return err
	}

	mux := http.NewServeMux()

	ztp.Register(mux, ztpConf)
	onie.Register(mux, onieImagesDir, onieConf)

	return http.ListenAndServe(listenAddr, mux)
}
