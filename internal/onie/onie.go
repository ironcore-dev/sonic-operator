// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package onie

import "net/http"

// Register a handler which serves the given directory over HTTP. See the ONIE
// documentation for which file names are tried:
// https://opencomputeproject.github.io/onie/design-spec/discovery.html
func Register(mux *http.ServeMux, installerDir string) {
	mux.Handle("GET /", http.FileServer(http.Dir(installerDir)))
}
