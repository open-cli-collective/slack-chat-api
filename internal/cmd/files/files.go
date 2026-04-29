package files

import "github.com/spf13/cobra"

// NewCmd returns the files command group
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "files",
		Short: "Manage Slack files",
	}

	cmd.AddCommand(newDownloadCmd())
	cmd.AddCommand(newGetCmd())

	return cmd
}
