package keychain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	serviceName  = "slack-chat-api"
	apiTokenKey  = "api_token"
	userTokenKey = "user_token"
)

// GetAPIToken retrieves the Slack API token from keychain/config or environment
func GetAPIToken() (string, error) {
	// Try secure storage first (keychain on macOS, config file on Linux)
	token, err := getCredential(apiTokenKey)
	if err == nil && token != "" {
		return token, nil
	}

	// Fallback to environment variable
	token = os.Getenv("SLACK_API_TOKEN")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no API token found - run 'slck config set-token' or set SLACK_API_TOKEN")
}

// SetAPIToken stores the Slack API token
func SetAPIToken(token string) error {
	return setCredential(apiTokenKey, token)
}

// DeleteAPIToken removes the Slack API token
func DeleteAPIToken() error {
	return deleteCredential(apiTokenKey)
}

// IsSecureStorage returns true if using secure storage (macOS Keychain)
func IsSecureStorage() bool {
	return runtime.GOOS == "darwin"
}

// HasStoredToken returns true if a token is stored in keychain/config (not env var)
func HasStoredToken() bool {
	token, err := getCredential(apiTokenKey)
	return err == nil && token != ""
}

// GetTokenSource returns where the current token is stored
func GetTokenSource() string {
	if token, err := getCredential(apiTokenKey); err == nil && token != "" {
		if runtime.GOOS == "darwin" {
			return "Keychain"
		}
		return "config file"
	}
	if os.Getenv("SLACK_API_TOKEN") != "" {
		return "environment variable"
	}
	return ""
}

// --- User Token (for search) ---

// GetUserToken retrieves the user token from keychain/config or environment
func GetUserToken() (string, error) {
	// Check environment variable first
	if token := os.Getenv("SLACK_USER_TOKEN"); token != "" {
		return token, nil
	}

	// Try secure storage (keychain on macOS, config file on Linux)
	token, err := getCredential(userTokenKey)
	if err == nil && token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no user token found - run 'slck config set-token <xoxp-token>' or set SLACK_USER_TOKEN")
}

// SetUserToken stores a user token
func SetUserToken(token string) error {
	return setCredential(userTokenKey, token)
}

// DeleteUserToken removes the stored user token
func DeleteUserToken() error {
	return deleteCredential(userTokenKey)
}

// HasStoredUserToken returns true if a user token is stored in keychain/config (not env var)
func HasStoredUserToken() bool {
	token, err := getCredential(userTokenKey)
	return err == nil && token != ""
}

// GetUserTokenSource returns where the user token comes from
func GetUserTokenSource() string {
	if token, err := getCredential(userTokenKey); err == nil && token != "" {
		if runtime.GOOS == "darwin" {
			return "Keychain"
		}
		return "config file"
	}
	if os.Getenv("SLACK_USER_TOKEN") != "" {
		return "environment variable"
	}
	return ""
}

// DetectTokenType returns "bot" for xoxb-*, "user" for xoxp-*, or "unknown"
func DetectTokenType(token string) string {
	if strings.HasPrefix(token, "xoxb-") {
		return "bot"
	}
	if strings.HasPrefix(token, "xoxp-") {
		return "user"
	}
	return "unknown"
}

// --- Platform-specific implementations ---

func getCredential(key string) (string, error) {
	if runtime.GOOS == "darwin" {
		return getFromKeychain(key)
	}
	return getFromConfigFile(key)
}

func setCredential(key, value string) error {
	if runtime.GOOS == "darwin" {
		return setInKeychain(key, value)
	}
	return setInConfigFile(key, value)
}

func deleteCredential(key string) error {
	if runtime.GOOS == "darwin" {
		return deleteFromKeychain(key)
	}
	return deleteFromConfigFile(key)
}

// --- macOS Keychain ---

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

// --- Config File (Linux fallback) ---

func getConfigDir() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "slack-chat-api")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "slack-chat-api")
}

func getConfigFilePath() string {
	return filepath.Join(getConfigDir(), "credentials")
}

func getFromConfigFile(key string) (string, error) {
	data, err := os.ReadFile(getConfigFilePath())
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("key not found")
}

func setInConfigFile(key, value string) error {
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	configPath := getConfigFilePath()

	// Read existing config
	existing := make(map[string]string)
	if data, err := os.ReadFile(configPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existing[parts[0]] = parts[1]
			}
		}
	}

	// Update value
	existing[key] = value

	// Write back
	var lines []string
	for k, v := range existing {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}

	return os.WriteFile(configPath, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func deleteFromConfigFile(key string) error {
	configPath := getConfigFilePath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var newLines []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && parts[0] != key {
			newLines = append(newLines, line)
		}
	}

	if len(newLines) == 0 {
		return os.Remove(configPath)
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")+"\n"), 0600)
}
