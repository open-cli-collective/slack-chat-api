package whoami

import (
	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
)

type whoamiOptions struct{}

// WhoamiResult represents the authenticated identity info
type WhoamiResult struct {
	Bot       *BotInfo       `json:"bot,omitempty"`
	User      *UserInfo      `json:"user,omitempty"`
	Workspace *WorkspaceInfo `json:"workspace"`
}

// BotInfo represents bot token identity
type BotInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// UserInfo represents user token identity
type UserInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// WorkspaceInfo represents workspace identity
type WorkspaceInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// NewCmd creates the whoami command
func NewCmd() *cobra.Command {
	opts := &whoamiOptions{}

	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the authenticated identity",
		Long: `Show the identity associated with the configured API tokens.

This is a quick way to verify which bot or user account will be used
for operations, and which workspace the tokens are associated with.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhoami(opts, nil, nil)
		},
	}
}

func runWhoami(opts *whoamiOptions, botClient *client.Client, userClient *client.Client) error {
	result := &WhoamiResult{}
	var workspace *WorkspaceInfo

	// Test bot token
	if botClient == nil {
		botClient, _ = client.New()
	}
	if botClient != nil {
		info, err := botClient.AuthTest()
		if err == nil {
			result.Bot = &BotInfo{
				Name: info.User,
				ID:   info.BotID,
			}
			if workspace == nil {
				workspace = &WorkspaceInfo{
					Name: info.Team,
					ID:   info.TeamID,
				}
			}
		}
	}

	// Test user token
	if userClient == nil {
		userClient, _ = client.NewUserClient()
	}
	if userClient != nil {
		info, err := userClient.AuthTest()
		if err == nil {
			result.User = &UserInfo{
				Name: info.User,
				ID:   info.UserID,
			}
			if workspace == nil {
				workspace = &WorkspaceInfo{
					Name: info.Team,
					ID:   info.TeamID,
				}
			}
		}
	}

	result.Workspace = workspace

	// Check if any token worked
	if result.Bot == nil && result.User == nil {
		output.Println("No valid tokens configured.")
		output.Println("Run 'slck config set-token' to configure authentication.")
		return nil
	}

	if output.IsJSON() {
		return output.PrintJSON(result)
	}

	// Text output
	if result.Bot != nil {
		if result.Bot.ID != "" {
			output.Printf("Bot: %s (%s)\n", result.Bot.Name, result.Bot.ID)
		} else {
			output.Printf("Bot: %s\n", result.Bot.Name)
		}
	}
	if result.User != nil {
		output.Printf("User: %s\n", result.User.Name)
	}
	if result.Workspace != nil {
		output.Printf("Workspace: %s\n", result.Workspace.Name)
	}

	return nil
}
