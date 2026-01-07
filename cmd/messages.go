package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/piekstra/slack-cli/internal/client"
	"github.com/spf13/cobra"
)

var messagesCmd = &cobra.Command{
	Use:     "messages",
	Aliases: []string{"msg", "m"},
	Short:   "Manage Slack messages",
}

var sendCmd = &cobra.Command{
	Use:   "send <channel> <text>",
	Short: "Send a message to a channel",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		threadTS, _ := cmd.Flags().GetString("thread")
		blocksJSON, _ := cmd.Flags().GetString("blocks")

		var blocks []interface{}
		if blocksJSON != "" {
			if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
				return fmt.Errorf("invalid blocks JSON: %w", err)
			}
		}

		msg, err := c.SendMessage(args[0], args[1], threadTS, blocks)
		if err != nil {
			return err
		}

		if outputJSON {
			data, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("Message sent (ts: %s)\n", msg.TS)
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update <channel> <timestamp> <text>",
	Short: "Update an existing message",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		blocksJSON, _ := cmd.Flags().GetString("blocks")

		var blocks []interface{}
		if blocksJSON != "" {
			if err := json.Unmarshal([]byte(blocksJSON), &blocks); err != nil {
				return fmt.Errorf("invalid blocks JSON: %w", err)
			}
		}

		if err := c.UpdateMessage(args[0], args[1], args[2], blocks); err != nil {
			return err
		}

		fmt.Println("Message updated")
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <channel> <timestamp>",
	Short: "Delete a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		if err := c.DeleteMessage(args[0], args[1]); err != nil {
			return err
		}

		fmt.Println("Message deleted")
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history <channel>",
	Short: "Get channel message history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		oldest, _ := cmd.Flags().GetString("oldest")
		latest, _ := cmd.Flags().GetString("latest")

		messages, err := c.GetChannelHistory(args[0], limit, oldest, latest)
		if err != nil {
			return err
		}

		if outputJSON {
			data, _ := json.MarshalIndent(messages, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(messages) == 0 {
			fmt.Println("No messages found")
			return nil
		}

		for _, m := range messages {
			ts := formatTimestamp(m.TS)
			text := truncate(m.Text, 80)
			fmt.Printf("[%s] %s: %s\n", ts, m.User, text)
		}

		return nil
	},
}

var threadCmd = &cobra.Command{
	Use:   "thread <channel> <thread-ts>",
	Short: "Get thread replies",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")

		messages, err := c.GetThreadReplies(args[0], args[1], limit)
		if err != nil {
			return err
		}

		if outputJSON {
			data, _ := json.MarshalIndent(messages, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(messages) == 0 {
			fmt.Println("No replies found")
			return nil
		}

		for _, m := range messages {
			ts := formatTimestamp(m.TS)
			text := truncate(m.Text, 80)
			fmt.Printf("[%s] %s: %s\n", ts, m.User, text)
		}

		return nil
	},
}

var reactCmd = &cobra.Command{
	Use:   "react <channel> <timestamp> <emoji>",
	Short: "Add a reaction to a message",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		// Remove colons if present
		emoji := strings.Trim(args[2], ":")

		if err := c.AddReaction(args[0], args[1], emoji); err != nil {
			return err
		}

		fmt.Printf("Added :%s: reaction\n", emoji)
		return nil
	},
}

var unreactCmd = &cobra.Command{
	Use:   "unreact <channel> <timestamp> <emoji>",
	Short: "Remove a reaction from a message",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.New()
		if err != nil {
			return err
		}

		// Remove colons if present
		emoji := strings.Trim(args[2], ":")

		if err := c.RemoveReaction(args[0], args[1], emoji); err != nil {
			return err
		}

		fmt.Printf("Removed :%s: reaction\n", emoji)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(messagesCmd)

	messagesCmd.AddCommand(sendCmd)
	sendCmd.Flags().String("thread", "", "Thread timestamp for reply")
	sendCmd.Flags().String("blocks", "", "Block Kit blocks as JSON array")

	messagesCmd.AddCommand(updateCmd)
	updateCmd.Flags().String("blocks", "", "Block Kit blocks as JSON array")
	messagesCmd.AddCommand(deleteCmd)

	messagesCmd.AddCommand(historyCmd)
	historyCmd.Flags().Int("limit", 20, "Maximum messages to return")
	historyCmd.Flags().String("oldest", "", "Only messages after this timestamp")
	historyCmd.Flags().String("latest", "", "Only messages before this timestamp")

	messagesCmd.AddCommand(threadCmd)
	threadCmd.Flags().Int("limit", 100, "Maximum replies to return")

	messagesCmd.AddCommand(reactCmd)
	messagesCmd.AddCommand(unreactCmd)
}

func formatTimestamp(ts string) string {
	// Slack timestamps are Unix timestamps with decimals
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return ts
	}

	var sec int64
	fmt.Sscanf(parts[0], "%d", &sec)
	t := time.Unix(sec, 0)
	return t.Format("2006-01-02 15:04")
}

func truncate(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
