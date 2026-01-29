package messages

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type updateOptions struct {
	blocksJSON string
	simple     bool
	noUnfurl   bool
}

func newUpdateCmd() *cobra.Command {
	opts := &updateOptions{}

	cmd := &cobra.Command{
		Use:   "update <channel> <timestamp> <text>",
		Short: "Update an existing message",
		Long: `Update an existing message.

By default, messages are updated using Slack Block Kit formatting for a more
refined appearance. Use --simple to update with plain text instead.`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(args[0], args[1], args[2], opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.blocksJSON, "blocks", "", "Block Kit blocks as JSON array (overrides default block formatting)")
	cmd.Flags().BoolVar(&opts.simple, "simple", false, "Update as plain text without block formatting")
	cmd.Flags().BoolVar(&opts.noUnfurl, "no-unfurl", false, "Disable link preview unfurling")

	return cmd
}

func runUpdate(channel, timestamp, text string, opts *updateOptions, c *client.Client) error {
	// Unescape shell-escaped characters (e.g., \! from zsh)
	text = unescapeShellChars(text)

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

	var blocks []interface{}
	if opts.blocksJSON != "" {
		if err := json.Unmarshal([]byte(opts.blocksJSON), &blocks); err != nil {
			return fmt.Errorf("invalid blocks JSON: %w", err)
		}
	} else if !opts.simple {
		// Default to block style for a more refined appearance
		blocks = buildDefaultBlocks(text)
	}

	if err := c.UpdateMessage(channelID, timestamp, text, blocks, !opts.noUnfurl); err != nil {
		return err
	}

	output.Println("Message updated")
	return nil
}
