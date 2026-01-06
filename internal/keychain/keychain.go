package keychain

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	serviceName = "slack-cli"
	apiTokenKey = "api_token"
)

// GetAPIToken retrieves the Slack API token from keychain or environment
func GetAPIToken() (string, error) {
	// Try keychain first
	token, err := getFromKeychain(apiTokenKey)
	if err == nil && token != "" {
		return token, nil
	}

	// Fallback to environment variable
	token = os.Getenv("SLACK_API_TOKEN")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no API token found - run 'slack-cli config set-token' or set SLACK_API_TOKEN")
}

// SetAPIToken stores the Slack API token in keychain
func SetAPIToken(token string) error {
	return setInKeychain(apiTokenKey, token)
}

// DeleteAPIToken removes the Slack API token from keychain
func DeleteAPIToken() error {
	return deleteFromKeychain(apiTokenKey)
}

func getFromKeychain(account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", serviceName,
		"-a", account,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func setInKeychain(account, value string) error {
	// First try to delete any existing item (ignore errors)
	_ = deleteFromKeychain(account)

	cmd := exec.Command("security", "add-generic-password",
		"-s", serviceName,
		"-a", account,
		"-w", value,
		"-U") // Update if exists

	return cmd.Run()
}

func deleteFromKeychain(account string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", account)

	return cmd.Run()
}
