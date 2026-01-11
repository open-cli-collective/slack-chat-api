package messages

import (
	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type deleteOptions struct{}

func newDeleteCmd() *cobra.Command {
	opts := &deleteOptions{}

	return &cobra.Command{
		Use:   "delete <channel> <timestamp>",
		Short: "Delete a message",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], args[1], opts, nil)
		},
	}
}

func runDelete(channel, timestamp string, opts *deleteOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.DeleteMessage(channel, timestamp); err != nil {
		return err
	}

	output.Println("Message deleted")
	return nil
}
