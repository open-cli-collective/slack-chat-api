package config

import (
	"github.com/spf13/cobra"
)

// NewCmd creates the config command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure slck",
	}

	cmd.AddCommand(newSetTokenCmd())
	cmd.AddCommand(newDeleteTokenCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newClearCmd())

	return cmd
}
