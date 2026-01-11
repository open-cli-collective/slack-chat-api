package channels

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type createOptions struct {
	private bool
}

func newCreateCmd() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], opts, nil)
		},
	}

	cmd.Flags().BoolVar(&opts.private, "private", false, "Create as private channel")

	return cmd
}

func runCreate(name string, opts *createOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	channel, err := c.CreateChannel(name, opts.private)
	if err != nil {
		return err
	}

	if output.JSON {
		data, _ := json.MarshalIndent(channel, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Created channel: %s (%s)\n", channel.Name, channel.ID)
	return nil
}
