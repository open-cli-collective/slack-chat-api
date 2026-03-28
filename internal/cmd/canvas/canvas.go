package canvas

import (
	"github.com/spf13/cobra"
)

// NewCmd creates the canvas command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "canvas",
		Aliases: []string{"cv"},
		Short:   "Manage Slack canvases",
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}
