package keychain

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetAPIToken_FromEnvVar(t *testing.T) {
	// Clear any existing env var first
	originalValue := os.Getenv("SLACK_API_TOKEN")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("SLACK_API_TOKEN", originalValue)
		} else {
			_ = os.Unsetenv("SLACK_API_TOKEN")
		}
	}()

	t.Setenv("SLACK_API_TOKEN", "xoxb-test-token-from-env")

	token, err := GetAPIToken()
	if err != nil {
		// On macOS, keychain might have a token which takes precedence
		// So we only check the env var path on non-darwin or when keychain fails
		if runtime.GOOS != "darwin" {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// If we got a token and we're not on darwin (where keychain takes precedence)
	if runtime.GOOS != "darwin" && token != "xoxb-test-token-from-env" {
		t.Errorf("expected token from env, got %s", token)
	}
}

func TestGetAPIToken_NoToken(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain may have stored token")
	}

	// Clear env var
	originalValue := os.Getenv("SLACK_API_TOKEN")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("SLACK_API_TOKEN", originalValue)
		} else {
			_ = os.Unsetenv("SLACK_API_TOKEN")
		}
	}()
	_ = os.Unsetenv("SLACK_API_TOKEN")

	// Use a temp dir that doesn't have any config
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := GetAPIToken()
	if err == nil {
		t.Error("expected error when no token is configured")
	}
}

func TestIsSecureStorage(t *testing.T) {
	expected := runtime.GOOS == "darwin"
	actual := IsSecureStorage()

	if actual != expected {
		t.Errorf("IsSecureStorage() = %v, expected %v", actual, expected)
	}
}

func TestGetConfigDir_Default(t *testing.T) {
	// Clear XDG_CONFIG_HOME to test default
	originalValue := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalValue)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "slack-cli")
	actual := getConfigDir()

	if actual != expected {
		t.Errorf("getConfigDir() = %s, expected %s", actual, expected)
	}
}

func TestGetConfigDir_XDGConfigHome(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	expected := filepath.Join(tmpDir, "slack-cli")
	actual := getConfigDir()

	if actual != expected {
		t.Errorf("getConfigDir() = %s, expected %s", actual, expected)
	}
}

func TestConfigFile_SetAndGet(t *testing.T) {
	// This tests the config file functions directly
	// which work on all platforms (used as fallback on macOS)

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test set
	err := setInConfigFile("test_key", "test_value")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Verify config directory was created
	configDir := filepath.Join(tmpDir, "slack-cli")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected config path to be a directory")
	}

	// Test get
	value, err := getFromConfigFile("test_key")
	if err != nil {
		t.Fatalf("getFromConfigFile failed: %v", err)
	}
	if value != "test_value" {
		t.Errorf("expected test_value, got %s", value)
	}
}

func TestConfigFile_GetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := getFromConfigFile("nonexistent_key")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestConfigFile_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set a value first
	err := setInConfigFile("delete_test", "value")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Verify it exists
	_, err = getFromConfigFile("delete_test")
	if err != nil {
		t.Fatalf("key not found after set: %v", err)
	}

	// Delete it
	err = deleteFromConfigFile("delete_test")
	if err != nil {
		t.Fatalf("deleteFromConfigFile failed: %v", err)
	}

	// Verify it's gone
	_, err = getFromConfigFile("delete_test")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestConfigFile_MultipleKeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set multiple keys
	err := setInConfigFile("key1", "value1")
	if err != nil {
		t.Fatalf("setInConfigFile key1 failed: %v", err)
	}
	err = setInConfigFile("key2", "value2")
	if err != nil {
		t.Fatalf("setInConfigFile key2 failed: %v", err)
	}
	err = setInConfigFile("key3", "value3")
	if err != nil {
		t.Fatalf("setInConfigFile key3 failed: %v", err)
	}

	// Verify all exist
	v1, _ := getFromConfigFile("key1")
	v2, _ := getFromConfigFile("key2")
	v3, _ := getFromConfigFile("key3")

	if v1 != "value1" || v2 != "value2" || v3 != "value3" {
		t.Errorf("multiple keys not stored correctly: %s, %s, %s", v1, v2, v3)
	}

	// Delete middle key
	err = deleteFromConfigFile("key2")
	if err != nil {
		t.Fatalf("deleteFromConfigFile failed: %v", err)
	}

	// Verify key1 and key3 still exist
	v1, err = getFromConfigFile("key1")
	if err != nil || v1 != "value1" {
		t.Errorf("key1 not preserved after deleting key2")
	}
	v3, err = getFromConfigFile("key3")
	if err != nil || v3 != "value3" {
		t.Errorf("key3 not preserved after deleting key2")
	}

	// Verify key2 is gone
	_, err = getFromConfigFile("key2")
	if err == nil {
		t.Error("key2 should be deleted")
	}
}

func TestConfigFile_UpdateExistingKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set initial value
	err := setInConfigFile("update_test", "original")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Update value
	err = setInConfigFile("update_test", "updated")
	if err != nil {
		t.Fatalf("setInConfigFile update failed: %v", err)
	}

	// Verify updated value
	value, err := getFromConfigFile("update_test")
	if err != nil {
		t.Fatalf("getFromConfigFile failed: %v", err)
	}
	if value != "updated" {
		t.Errorf("expected updated, got %s", value)
	}
}

func TestConfigFile_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("File permissions test not applicable on Windows")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set a value to create the file
	err := setInConfigFile("perm_test", "value")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Check file permissions (should be 0600)
	configPath := filepath.Join(tmpDir, "slack-cli", "credentials")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("expected file permissions 0600, got %o", mode)
	}

	// Check directory permissions (should be 0700)
	dirInfo, err := os.Stat(filepath.Join(tmpDir, "slack-cli"))
	if err != nil {
		t.Fatalf("stat dir failed: %v", err)
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode != 0700 {
		t.Errorf("expected directory permissions 0700, got %o", dirMode)
	}
}

func TestConfigFile_ValueWithEquals(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test value containing equals sign
	valueWithEquals := "token=part=with=equals"
	err := setInConfigFile("equals_test", valueWithEquals)
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	value, err := getFromConfigFile("equals_test")
	if err != nil {
		t.Fatalf("getFromConfigFile failed: %v", err)
	}
	if value != valueWithEquals {
		t.Errorf("expected %s, got %s", valueWithEquals, value)
	}
}

func TestHasStoredToken_WithToken(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Initially no token
	if HasStoredToken() {
		t.Error("expected no stored token initially")
	}

	// Set a token
	err := setInConfigFile(apiTokenKey, "xoxb-test-token")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Now should have a token
	if !HasStoredToken() {
		t.Error("expected stored token after set")
	}
}

func TestHasStoredToken_EnvVarOnly(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("SLACK_API_TOKEN", "xoxb-env-token")

	// HasStoredToken should return false when only env var is set
	if HasStoredToken() {
		t.Error("expected false when only env var is set")
	}
}

func TestGetTokenSource_ConfigFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Clear env var
	originalValue := os.Getenv("SLACK_API_TOKEN")
	defer func() {
		if originalValue != "" {
			_ = os.Setenv("SLACK_API_TOKEN", originalValue)
		} else {
			_ = os.Unsetenv("SLACK_API_TOKEN")
		}
	}()
	_ = os.Unsetenv("SLACK_API_TOKEN")

	// No token - should return empty string
	source := GetTokenSource()
	if source != "" {
		t.Errorf("expected empty string for no token, got %s", source)
	}

	// Set a token in config file
	err := setInConfigFile(apiTokenKey, "xoxb-test-token")
	if err != nil {
		t.Fatalf("setInConfigFile failed: %v", err)
	}

	// Should return "config file" on non-darwin
	source = GetTokenSource()
	if source != "config file" {
		t.Errorf("expected 'config file', got %s", source)
	}
}

func TestGetTokenSource_EnvVar(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("SLACK_API_TOKEN", "xoxb-env-token")

	// When only env var is set (no config file)
	source := GetTokenSource()
	if source != "environment variable" {
		t.Errorf("expected 'environment variable', got %s", source)
	}
}

// --- User Token Tests ---

func TestUserToken_SetAndGet(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Clear env var
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	// Set user token
	err := SetUserToken("xoxp-test-user-token")
	if err != nil {
		t.Fatalf("SetUserToken failed: %v", err)
	}

	// Get user token
	token, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken failed: %v", err)
	}
	if token != "xoxp-test-user-token" {
		t.Errorf("expected xoxp-test-user-token, got %s", token)
	}
}

func TestUserToken_Delete(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	// Set a token first
	err := SetUserToken("xoxp-delete-test")
	if err != nil {
		t.Fatalf("SetUserToken failed: %v", err)
	}

	// Verify it exists
	if !HasStoredUserToken() {
		t.Error("expected stored user token after set")
	}

	// Delete it
	err = DeleteUserToken()
	if err != nil {
		t.Fatalf("DeleteUserToken failed: %v", err)
	}

	// Verify it's gone
	if HasStoredUserToken() {
		t.Error("expected no stored user token after delete")
	}
}

func TestUserToken_EnvVar(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("SLACK_USER_TOKEN", "xoxp-env-user-token")

	// Should get token from env var
	token, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken failed: %v", err)
	}
	if token != "xoxp-env-user-token" {
		t.Errorf("expected xoxp-env-user-token, got %s", token)
	}
}

func TestUserToken_NoToken(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	_, err := GetUserToken()
	if err == nil {
		t.Error("expected error when no user token is configured")
	}
}

func TestHasStoredUserToken(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	// Initially no token
	if HasStoredUserToken() {
		t.Error("expected no stored user token initially")
	}

	// Set a token
	err := SetUserToken("xoxp-test-token")
	if err != nil {
		t.Fatalf("SetUserToken failed: %v", err)
	}

	// Now should have a token
	if !HasStoredUserToken() {
		t.Error("expected stored user token after set")
	}
}

func TestHasStoredUserToken_EnvVarOnly(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("SLACK_USER_TOKEN", "xoxp-env-token")

	// HasStoredUserToken should return false when only env var is set
	if HasStoredUserToken() {
		t.Error("expected false when only env var is set")
	}
}

func TestGetUserTokenSource_ConfigFile(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	// No token - should return empty string
	source := GetUserTokenSource()
	if source != "" {
		t.Errorf("expected empty string for no token, got %s", source)
	}

	// Set a token in config file
	err := SetUserToken("xoxp-test-token")
	if err != nil {
		t.Fatalf("SetUserToken failed: %v", err)
	}

	// Should return "config file" on non-darwin
	source = GetUserTokenSource()
	if source != "config file" {
		t.Errorf("expected 'config file', got %s", source)
	}
}

func TestGetUserTokenSource_EnvVar(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv("SLACK_USER_TOKEN", "xoxp-env-token")

	// When only env var is set (no config file)
	source := GetUserTokenSource()
	if source != "environment variable" {
		t.Errorf("expected 'environment variable', got %s", source)
	}
}

func TestDetectTokenType(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"xoxb-123-456-789", "bot"},
		{"xoxp-123-456-789", "user"},
		{"invalid-token", "unknown"},
		{"", "unknown"},
		{"xoxb", "unknown"},
		{"xoxp", "unknown"},
		{"xoxb-", "bot"},
		{"xoxp-", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := DetectTokenType(tt.token)
			if result != tt.expected {
				t.Errorf("DetectTokenType(%q) = %q, expected %q", tt.token, result, tt.expected)
			}
		})
	}
}

func TestBothTokenTypes_Coexist(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping on macOS - keychain tests require manual setup")
	}

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	_ = os.Unsetenv("SLACK_API_TOKEN")
	_ = os.Unsetenv("SLACK_USER_TOKEN")

	// Set both tokens
	err := SetAPIToken("xoxb-bot-token")
	if err != nil {
		t.Fatalf("SetAPIToken failed: %v", err)
	}
	err = SetUserToken("xoxp-user-token")
	if err != nil {
		t.Fatalf("SetUserToken failed: %v", err)
	}

	// Verify both exist independently
	botToken, err := GetAPIToken()
	if err != nil {
		t.Fatalf("GetAPIToken failed: %v", err)
	}
	if botToken != "xoxb-bot-token" {
		t.Errorf("expected xoxb-bot-token, got %s", botToken)
	}

	userToken, err := GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken failed: %v", err)
	}
	if userToken != "xoxp-user-token" {
		t.Errorf("expected xoxp-user-token, got %s", userToken)
	}

	// Delete bot token, user token should remain
	err = DeleteAPIToken()
	if err != nil {
		t.Fatalf("DeleteAPIToken failed: %v", err)
	}

	// User token should still exist
	userToken, err = GetUserToken()
	if err != nil {
		t.Fatalf("GetUserToken after bot delete failed: %v", err)
	}
	if userToken != "xoxp-user-token" {
		t.Errorf("user token should persist after bot token delete")
	}

	// Bot token should be gone
	if HasStoredToken() {
		t.Error("bot token should be deleted")
	}
}
