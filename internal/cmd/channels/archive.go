package channels

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

type archiveOptions struct {
	force bool
	stdin io.Reader // For testing
}

func newArchiveCmd() *cobra.Command {
	opts := &archiveOptions{}

	cmd := &cobra.Command{
		Use:   "archive <channel-id>",
		Short: "Archive a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(args[0], opts, nil)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runArchive(channelID string, opts *archiveOptions, c *client.Client) error {
	// Validate channel ID
	if err := validate.ChannelID(channelID); err != nil {
		return err
	}

	// Prompt for confirmation unless --force
	if !opts.force {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}

		output.Printf("About to archive channel: %s\n", channelID)
		output.Printf("Are you sure? [y/N]: ")

		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			confirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if confirm != "y" && confirm != "yes" {
				output.Println("Cancelled.")
				return nil
			}
		}
	}

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.ArchiveChannel(channelID); err != nil {
		return client.WrapError(fmt.Sprintf("archive channel %s", channelID), err)
	}

	output.Printf("Archived channel: %s\n", channelID)
	return nil
}
