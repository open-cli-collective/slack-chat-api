package files

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <file-id-or-url>",
		Short: "Get file metadata",
		Long: `Get compact plaintext metadata for a Slack file.

Accepts a file ID (e.g. F0AHF3NUSQK) or a Slack file URL
(url_private, url_private_download, or permalink).

Pairs with 'slck files download <ref>' for the search-to-get/download
workflow: refs from 'slck search files' can be passed directly.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGet(args[0], nil)
		},
	}
	return cmd
}

func runGet(input string, c *client.Client) error {
	fileID := resolveFileID(input)
	if fileID == "" {
		return fmt.Errorf("could not resolve file ID from %q — provide a file ID (e.g. F0AHF3NUSQK) or Slack file URL", input)
	}

	if c == nil {
		var err error
		c, err = client.New()
		if err != nil {
			return err
		}
	}

	info, err := c.GetFileInfo(fileID)
	if err != nil {
		return err
	}

	if output.IsJSON() {
		return output.PrintJSON(info)
	}

	output.Printf("ID: %s\n", info.ID)
	output.Printf("Name: %s\n", info.Name)
	if info.Title != "" {
		output.Printf("Title: %s\n", info.Title)
	}
	if info.Filetype != "" {
		output.Printf("Type: %s\n", info.Filetype)
	}
	output.Printf("Size: %s\n", output.HumanSize(info.Size))
	output.Printf("Download: slck files download %s\n", info.ID)

	return nil
}
