package messages

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

type deleteOptions struct {
	force bool
	stdin io.Reader // For testing
}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	cmd := &cobra.Command{
		Use:   "delete <channel> <timestamp>",
		Short: "Delete a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], args[1], opts, nil)
		},
	}

	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(channel, timestamp string, opts *deleteOptions, c *client.Client) error {
	// Validate timestamp
	if err := validate.Timestamp(timestamp); err != nil {
		return err
	}

	// Prompt for confirmation unless --force
	if !opts.force {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}

		output.Printf("About to delete message %s in channel %s\n", timestamp, channel)
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

	if err := c.DeleteMessage(channelID, timestamp); err != nil {
		return client.WrapError(fmt.Sprintf("delete message %s", timestamp), err)
	}

	output.Println("Message deleted")
	return nil
}
