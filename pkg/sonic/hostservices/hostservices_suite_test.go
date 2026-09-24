// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package hostservices

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHostServices(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HostServices Suite")
}
