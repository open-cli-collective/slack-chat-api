package messages

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/piekstra/slack-cli/internal/output"
)

type sendOptions struct {
	threadTS   string
	blocksJSON string
	simple     bool
}

func newSendCmd() *cobra.Command {
	opts := &sendOptions{}

	cmd := &cobra.Command{
		Use:   "send <channel> <text>",
		Short: "Send a message to a channel",
		Long: `Send a message to a channel.

By default, messages are sent using Slack Block Kit formatting for a more
refined appearance. Use --simple to send plain text messages instead.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSend(args[0], args[1], opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.threadTS, "thread", "", "Thread timestamp for reply")
	cmd.Flags().StringVar(&opts.blocksJSON, "blocks", "", "Block Kit blocks as JSON array (overrides default block formatting)")
	cmd.Flags().BoolVar(&opts.simple, "simple", false, "Send as plain text without block formatting")

	return cmd
}

func runSend(channel, text string, opts *sendOptions, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
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

	msg, err := c.SendMessage(channel, text, opts.threadTS, blocks)
	if err != nil {
		return err
	}

	if output.JSON {
		data, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Message sent (ts: %s)\n", msg.TS)
	return nil
}
