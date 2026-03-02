package files

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

// fileIDPattern matches Slack file IDs (e.g. F0AHF3NUSQK)
var fileIDPattern = regexp.MustCompile(`^F[A-Z0-9]+$`)

// urlFileIDPattern extracts file IDs from Slack URLs
// Matches patterns like /files/U.../F0AHF3NUSQK/... or /files-pri/T...-F0AHF3NUSQK/...
var urlFileIDPattern = regexp.MustCompile(`[/-](F[A-Z0-9]+)[/]`)

type downloadOptions struct {
	outputPath string
}

func newDownloadCmd() *cobra.Command {
	opts := &downloadOptions{}

	cmd := &cobra.Command{
		Use:   "download <file-id-or-url>",
		Short: "Download a Slack file",
		Long: `Download a file from Slack by file ID or URL.

Accepts a file ID (e.g. F0AHF3NUSQK) or a Slack file URL
(url_private, url_private_download, or permalink).

By default saves to the current directory using the file's original name.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownload(args[0], opts, nil)
		},
	}

	cmd.Flags().StringVarP(&opts.outputPath, "output", "O", "", "Output file path (default: ./<filename>)")

	return cmd
}

func runDownload(input string, opts *downloadOptions, c *client.Client) error {
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

	downloadURL := info.URLPrivateDownload
	if downloadURL == "" {
		downloadURL = info.URLPrivate
	}
	if downloadURL == "" {
		return fmt.Errorf("no download URL available for file %s", fileID)
	}

	destPath := opts.outputPath
	if destPath == "" {
		destPath = info.Name
	}

	if output.IsJSON() {
		return output.PrintJSON(map[string]interface{}{
			"file_id": info.ID,
			"name":    info.Name,
			"size":    info.Size,
			"path":    destPath,
		})
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := c.DownloadFile(downloadURL, f); err != nil {
		// Clean up partial file on error
		_ = os.Remove(destPath)
		return fmt.Errorf("downloading file: %w", err)
	}

	absPath, _ := filepath.Abs(destPath)
	output.Printf("Downloaded %s (%d bytes) to %s\n", info.Name, info.Size, absPath)

	return nil
}

// resolveFileID extracts a Slack file ID from a raw ID or URL
func resolveFileID(input string) string {
	if fileIDPattern.MatchString(input) {
		return input
	}

	matches := urlFileIDPattern.FindStringSubmatch(input)
	if len(matches) >= 2 {
		return matches[1]
	}

	return ""
}
