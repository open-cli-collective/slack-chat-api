package canvas

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type editOptions struct {
	text  string
	file  string
	stdin io.Reader // For testing
}

func newEditCmd() *cobra.Command {
	opts := &editOptions{}

	cmd := &cobra.Command{
		Use:   "edit <canvas-id>",
		Short: "Edit a canvas",
		Long: `Replace the content of an existing canvas with new markdown.

  slck canvas edit F12345 --text "# Updated content"
  slck canvas edit F12345 --file updated.md
  echo "# New content" | slck canvas edit F12345 --file -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(args[0], opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.text, "text", "", "Markdown content as inline text")
	cmd.Flags().StringVar(&opts.file, "file", "", "Read markdown content from file (use - for stdin)")

	return cmd
}

func runEdit(canvasID string, opts *editOptions, c *client.Client) error {
	markdown, err := resolveContent(opts.text, opts.file, opts.stdin)
	if err != nil {
		return err
	}
	if markdown == "" {
		return fmt.Errorf("content required: use --text, --file, or --file -")
	}

	if c == nil {
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.EditCanvas(canvasID, markdown); err != nil {
		return client.WrapError("edit canvas", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]string{"canvas_id": canvasID, "status": "updated"})
	}
	output.Printf("Updated canvas: %s\n", canvasID)
	return nil
}
