package messages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/piekstra/slack-chat-api/internal/client"
	"github.com/piekstra/slack-chat-api/internal/output"
	"github.com/piekstra/slack-chat-api/internal/validate"
)

type sendOptions struct {
	threadTS   string
	blocksJSON string
	simple     bool
	stdin      io.Reader // For testing
}

func newSendCmd() *cobra.Command {
	opts := &sendOptions{}

	cmd := &cobra.Command{
		Use:   "send <channel> <text>",
		Short: "Send a message to a channel",
		Long: `Send a message to a channel.

By default, messages are sent using Slack Block Kit formatting for a more
refined appearance. Use --simple to send plain text messages instead.

Use "-" as the text argument to read from stdin:
  echo "Hello" | slack-chat-api messages send C1234567890 -
  cat message.txt | slack-chat-api messages send C1234567890 -`,
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
	// Validate channel ID
	if err := validate.ChannelID(channel); err != nil {
		return err
	}

	// Validate thread timestamp if provided
	if opts.threadTS != "" {
		if err := validate.Timestamp(opts.threadTS); err != nil {
			return err
		}
	}

	// Read from stdin if text is "-"
	if text == "-" {
		reader := opts.stdin
		if reader == nil {
			reader = os.Stdin
		}
		scanner := bufio.NewScanner(reader)
		var lines []byte
		for scanner.Scan() {
			if len(lines) > 0 {
				lines = append(lines, '\n')
			}
			lines = append(lines, scanner.Bytes()...)
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		text = string(lines)
	}

	if text == "" {
		return fmt.Errorf("message text cannot be empty")
	}

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
		return client.WrapError("send message", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(msg)
	}

	output.Printf("Message sent (ts: %s)\n", msg.TS)
	return nil
}
