package canvas

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <canvas-id>",
		Short: "Delete a canvas",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args[0], nil)
		},
	}

	return cmd
}

func runDelete(canvasID string, c *client.Client) error {
	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	if err := c.DeleteCanvas(canvasID); err != nil {
		return client.WrapError("delete canvas", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]string{"canvas_id": canvasID, "status": "deleted"})
	}
	output.Printf("Deleted canvas: %s\n", canvasID)
	return nil
}
