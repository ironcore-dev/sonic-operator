// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FlagSpec struct {
	Name     string
	Usage    string
	Required bool
	BindFunc func(fs *pflag.FlagSet)
}

func AddFlags(fs *pflag.FlagSet, cmd *cobra.Command, specs []FlagSpec) error {
	for _, spec := range specs {
		spec.BindFunc(fs)
		if spec.Required {
			if err := cmd.MarkFlagRequired(spec.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
