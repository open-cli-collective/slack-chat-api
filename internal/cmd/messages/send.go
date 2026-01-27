package messages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/validate"
)

type sendOptions struct {
	threadTS    string
	blocksJSON  string
	blocksFile  string
	blocksStdin bool
	simple      bool
	noUnfurl    bool
	files       []string
	fileTitle   string
	stdin       io.Reader // For testing
}

func newSendCmd() *cobra.Command {
	opts := &sendOptions{}

	cmd := &cobra.Command{
		Use:   "send <channel> [text]",
		Short: "Send a message to a channel",
		Long: `Send a message to a channel.

By default, messages are sent using Slack Block Kit formatting for a more
refined appearance. Use --simple to send plain text messages instead.

Use "-" as the text argument to read message text from stdin:
  echo "Hello" | slck messages send C1234567890 -
  cat message.txt | slck messages send C1234567890 -

BLOCK KIT OPTIONS

Text is optional when providing blocks via any of these methods:

  --blocks        Inline JSON. Best for simple, single-line blocks.
                  Complex JSON requires careful shell escaping.

  --blocks-file   Read from a file. Recommended for multi-line or complex
                  Block Kit payloads - avoids shell escaping issues entirely.

  --blocks-stdin  Read from stdin. Useful for piping output from other tools
                  (e.g., jq, scripts) directly into Slack.

Examples:
  slck messages send C1234567890 --blocks '[{"type":"section",...}]'
  slck messages send C1234567890 --blocks-file ./report.json
  generate-report | slck messages send C1234567890 --blocks-stdin

FILE UPLOADS

  --file            Upload a file to the channel. Can be specified multiple
                    times to upload several files.
  --file-title      Custom title for uploaded file(s).

Examples:
  slck messages send C1234567890 --file ./report.pdf
  slck messages send C1234567890 "Here's the report" --file ./report.pdf
  slck messages send C1234567890 --file ./report.pdf --thread 1234567890.123456
  slck messages send C1234567890 --file ./report.pdf --file-title "Monthly Report"
  slck messages send C1234567890 --file ./a.csv --file ./b.csv`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := ""
			if len(args) > 1 {
				text = args[1]
			}
			return runSend(args[0], text, opts, nil)
		},
	}

	cmd.Flags().StringVar(&opts.threadTS, "thread", "", "Thread timestamp for reply")
	cmd.Flags().StringVar(&opts.blocksJSON, "blocks", "", "Inline Block Kit JSON array (for simple blocks)")
	cmd.Flags().StringVar(&opts.blocksFile, "blocks-file", "", "Read blocks from JSON file (recommended for complex payloads)")
	cmd.Flags().BoolVar(&opts.blocksStdin, "blocks-stdin", false, "Read blocks from stdin (for piping from other tools)")
	cmd.Flags().BoolVar(&opts.simple, "simple", false, "Send as plain text without block formatting")
	cmd.Flags().BoolVar(&opts.noUnfurl, "no-unfurl", false, "Disable link preview unfurling")
	cmd.Flags().StringArrayVar(&opts.files, "file", nil, "File(s) to upload (can be specified multiple times)")
	cmd.Flags().StringVar(&opts.fileTitle, "file-title", "", "Custom title for uploaded file(s)")

	return cmd
}

func runSend(channel, text string, opts *sendOptions, c *client.Client) error {
	// Validate and normalize thread timestamp if provided
	if opts.threadTS != "" {
		if err := validate.Timestamp(opts.threadTS); err != nil {
			return err
		}
		opts.threadTS = validate.NormalizeTimestamp(opts.threadTS)
	}

	// Validate mutually exclusive blocks options
	blocksOptionsCount := 0
	if opts.blocksJSON != "" {
		blocksOptionsCount++
	}
	if opts.blocksFile != "" {
		blocksOptionsCount++
	}
	if opts.blocksStdin {
		blocksOptionsCount++
	}
	if blocksOptionsCount > 1 {
		return fmt.Errorf("only one of --blocks, --blocks-file, or --blocks-stdin can be specified")
	}

	// Read from stdin if text is "-"
	if text == "-" {
		if opts.blocksStdin {
			return fmt.Errorf("cannot use '-' for text and --blocks-stdin together; stdin can only be used for one")
		}
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

	// Unescape shell-escaped characters (e.g., \! from zsh)
	text = unescapeShellChars(text)

	// Determine blocks source
	var blocksSource string
	if opts.blocksJSON != "" {
		blocksSource = opts.blocksJSON
	} else if opts.blocksFile != "" {
		data, err := os.ReadFile(opts.blocksFile)
		if err != nil {
			return fmt.Errorf("reading blocks file: %w", err)
		}
		blocksSource = string(data)
	} else if opts.blocksStdin {
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
			return fmt.Errorf("reading blocks from stdin: %w", err)
		}
		blocksSource = string(lines)
	}

	// Validate: must have text, blocks, or files
	hasBlocks := blocksSource != ""
	hasFiles := len(opts.files) > 0
	if text == "" && !hasBlocks && !hasFiles {
		return fmt.Errorf("message text cannot be empty (or provide blocks via --blocks, --blocks-file, --blocks-stdin, or files via --file)")
	}

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

	// Handle file uploads
	if hasFiles {
		return uploadFiles(c, channelID, text, opts)
	}

	var blocks []interface{}
	if blocksSource != "" {
		if err := json.Unmarshal([]byte(blocksSource), &blocks); err != nil {
			return fmt.Errorf("invalid blocks JSON: %w", err)
		}
	} else if !opts.simple && text != "" {
		// Default to block style for a more refined appearance
		blocks = buildDefaultBlocks(text)
	}

	msg, err := c.SendMessage(channelID, text, opts.threadTS, blocks, !opts.noUnfurl)
	if err != nil {
		return client.WrapError("send message", err)
	}

	if output.IsJSON() {
		return output.PrintJSON(msg)
	}

	output.Printf("Message sent (ts: %s)\n", msg.TS)
	return nil
}

func uploadFiles(c *client.Client, channelID, text string, opts *sendOptions) error {
	var uploadedFiles []client.CompleteUploadExternalFile

	for _, filePath := range opts.files {
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("cannot access file %s: %w", filePath, err)
		}

		filename := filepath.Base(filePath)
		output.Printf("Uploading %s (%d bytes)...\n", filename, info.Size())

		// Step 1: Get upload URL
		uploadResp, err := c.GetUploadURLExternal(filename, info.Size())
		if err != nil {
			return client.WrapError("get upload URL", err)
		}

		// Step 2: Upload file bytes
		f, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("opening file %s: %w", filePath, err)
		}

		err = c.UploadFileToURL(uploadResp.UploadURL, f)
		_ = f.Close()
		if err != nil {
			return client.WrapError("upload file", err)
		}

		title := opts.fileTitle
		if title == "" {
			title = filename
		}

		uploadedFiles = append(uploadedFiles, client.CompleteUploadExternalFile{
			ID:    uploadResp.FileID,
			Title: title,
		})
	}

	// Step 3: Complete upload and share to channel/thread
	err := c.CompleteUploadExternal(uploadedFiles, channelID, opts.threadTS, text)
	if err != nil {
		return client.WrapError("complete upload", err)
	}

	if len(uploadedFiles) == 1 {
		output.Printf("File uploaded to channel %s\n", channelID)
	} else {
		output.Printf("%d files uploaded to channel %s\n", len(uploadedFiles), channelID)
	}

	return nil
}
