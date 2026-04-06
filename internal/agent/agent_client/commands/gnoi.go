package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	client "github.com/ironcore-dev/sonic-operator/internal/agent/agent_client/client"
)

func Gnoi() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "gnoi [subcommand]",
		Args: cobra.NoArgs,
		RunE: SubcommandRequired,
	}

	subcommands := []*cobra.Command{
		SaveConfig(),
	}

	cmd.AddCommand(subcommands...)
	return cmd
}

func SaveConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "save-config",
		Short:   "Save the current configuration",
		Example: "agent_cli gnoi save-config",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunSaveConfig(cmd.Context(), GetSharedSwitchAgentClient())
		},
	}
	return cmd
}

func RunSaveConfig(
	ctx context.Context,
	c client.SwitchAgentClient,
) error {
	err := c.SaveConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, err = fmt.Fprintln(os.Stdout, "Config saved successfully")
	return err
}
