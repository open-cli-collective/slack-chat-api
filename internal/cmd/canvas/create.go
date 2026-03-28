package canvas

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type createOptions struct {
	title   string
	text    string
	file    string
	channel string
	stdin   io.Reader // For testing
}

func newCreateCmd() *cobra.Command {
	opts := &createOptions{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a canvas",
		Long: `Create a standalone or channel canvas with markdown content.

Provide content via --text, --file, or stdin:
  slck canvas create --title "Report" --text "# Heading"
  slck canvas create --title "Report" --file report.md
  echo "# Report" | slck canvas create --title "Report" --file -

Create a channel canvas (pinned to channel tab):
  slck canvas create --channel C1234567890 --file runbook.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.title, "title", "", "Canvas title (standalone canvases only)")
	cmd.Flags().StringVar(&opts.text, "text", "", "Markdown content as inline text")
	cmd.Flags().StringVar(&opts.file, "file", "", "Read markdown content from file (use - for stdin)")
	cmd.Flags().StringVar(&opts.channel, "channel", "", "Channel ID to create a channel canvas")

	return cmd
}

func runCreate(opts *createOptions, c *client.Client) error {
	markdown, err := resolveContent(opts.text, opts.file, opts.stdin)
	if err != nil {
		return err
	}
	if markdown == "" {
		return fmt.Errorf("content required: use --text, --file, or --file -")
	}

	// Validate flags before client init so users get clear errors without needing a valid token
	if opts.channel == "" && opts.title == "" {
		return fmt.Errorf("--title is required for standalone canvases")
	}
	if opts.channel != "" && opts.title != "" {
		return fmt.Errorf("--title is not used with --channel (channel canvases don't have titles)")
	}

	if c == nil {
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if opts.channel != "" {
		channelID, err := c.ResolveChannel(opts.channel)
		if err != nil {
			return err
		}
		canvasID, err := c.CreateChannelCanvas(channelID, markdown)
		if err != nil {
			return client.WrapError("create channel canvas", err)
		}
		if output.IsJSON() {
			return output.PrintJSON(map[string]string{"canvas_id": canvasID, "channel": channelID})
		}
		output.Printf("Created channel canvas: %s\n", canvasID)
		return nil
	}

	canvasID, err := c.CreateCanvas(opts.title, markdown)
	if err != nil {
		return client.WrapError("create canvas", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]string{"canvas_id": canvasID, "title": opts.title})
	}
	output.Printf("Created canvas: %s\n", canvasID)
	return nil
}

func resolveContent(text, file string, stdin io.Reader) (string, error) {
	if text != "" && file != "" {
		return "", fmt.Errorf("cannot use both --text and --file")
	}

	if text != "" {
		return text, nil
	}

	if file == "-" {
		return readStdin(stdin)
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

func readStdin(r io.Reader) (string, error) {
	if r == nil {
		r = os.Stdin
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	return string(data), nil
}
