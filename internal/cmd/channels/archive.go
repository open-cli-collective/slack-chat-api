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
)

type archiveOptions struct {
	force bool
	stdin io.Reader // For testing
}

func newArchiveCmd() *cobra.Command {
	opts := &archiveOptions{}

	cmd := &cobra.Command{
		Use:   "archive <channel>",
		Short: "Archive a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runArchive(args[0], opts, nil)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runArchive(channel string, opts *archiveOptions, c *client.Client) error {
	// Prompt for confirmation unless --force
	if !opts.force {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}

		output.Printf("About to archive channel: %s\n", channel)
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

	// Resolve channel name to ID if needed
	channelID, err := c.ResolveChannel(channel)
	if err != nil {
		return err
	}

	if err := c.ArchiveChannel(channelID); err != nil {
		return client.WrapError(fmt.Sprintf("archive channel %s", channel), err)
	}

	output.Printf("Archived channel: %s\n", channel)
	return nil
}
