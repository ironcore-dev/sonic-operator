// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package commands

// import (
// 	"context"
// 	"fmt"
// 	"os"

// 	"switch-proxy/internal/api"
// 	"switch-proxy/proxy_client/client"

// 	"github.com/spf13/cobra"
// )

// type DeleteVlanOptions struct {
// 	VlanName string
// }

// func DeleteVlan(printer client.PrintRenderer) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:     "vlan <vlan-name>",
// 		Short:   "Delete a VLAN",
// 		Example: "agent_cli delete vlan VLAN_100",
// 		Args:    cobra.ExactArgs(1),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			opts := DeleteVlanOptions{
// 				VlanName: args[0],
// 			}
// 			return RunDeleteVlan(cmd.Context(), GetSharedSwitchProxyClient(), printer, opts)
// 		},
// 	}
// 	return cmd
// }

// func RunDeleteVlan(
// 	ctx context.Context,
// 	c client.SwitchProxyClient,
// 	printer client.PrintRenderer,
// 	opts DeleteVlanOptions,
// ) error {
// 	fmt.Printf("Deleting VLAN %s\n", opts.VlanName)

// 	vlan, err := c.DeleteVlan(ctx, &api.Vlan{
// 		TypeMeta: api.TypeMeta{
// 			Kind: api.VlanKind,
// 		},
// 		VlanMeta: api.VlanMeta{
// 			Name: opts.VlanName,
// 		},
// 		Spec: api.VlanSpec{
// 		},
// 	})
// 	if err != nil {
// 		fmt.Println("Error occurred while deleting VLAN:", err)
// 		return fmt.Errorf("failed to delete VLAN: %w", err)
// 	}

// 	return printer.Print("VLAN deleted successfully", os.Stdout, vlan)
// }
