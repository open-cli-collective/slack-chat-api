package channels

import (
	"github.com/spf13/cobra"
)

// NewCmd creates the channels command with all subcommands
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channels",
		Aliases: []string{"ch"},
		Short:   "Manage Slack channels",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newArchiveCmd())
	cmd.AddCommand(newUnarchiveCmd())
	cmd.AddCommand(newSetTopicCmd())
	cmd.AddCommand(newSetPurposeCmd())
	cmd.AddCommand(newInviteCmd())
	cmd.AddCommand(newJoinCmd())
	cmd.AddCommand(newLeaveCmd())

	return cmd
}
