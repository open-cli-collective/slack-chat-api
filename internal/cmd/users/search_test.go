package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
)

// Helper to create a test client with a mock server
func newTestClient(handler http.HandlerFunc) (*client.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	c := client.NewWithConfig(server.URL, "xoxb-test-token", nil)
	return c, server
}

func TestRunSearchUsers_Success(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "john.doe",
				"real_name": "John Doe",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "john@example.com",
					"display_name": "Johnny",
				},
			},
			{
				"id":        "U456",
				"name":      "jane.doe",
				"real_name": "Jane Doe",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "jane@example.com",
					"display_name": "Janey",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users.list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "all",
	}

	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_NoResults(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "alice",
				"real_name": "Alice Smith",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "alice@example.com",
					"display_name": "Alice",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "all",
	}

	err := runSearch("nonexistent12345", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_ByName(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "john.doe",
				"real_name": "Alice Smith",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "alice@example.com",
					"display_name": "Alice",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "name",
	}

	// Should match because username is "john.doe"
	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_ByEmail(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "alice",
				"real_name": "Alice Smith",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "john@company.com",
					"display_name": "Alice",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "email",
	}

	// Should match because email contains "john"
	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_ByDisplayName(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "alice",
				"real_name": "Alice Smith",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "alice@example.com",
					"display_name": "Johnny Boy",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "display_name",
	}

	// Should match because display_name contains "Johnny"
	err := runSearch("johnny", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_CaseInsensitive(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "JOHN.DOE",
				"real_name": "John Doe",
				"is_bot":    false,
				"profile": map[string]interface{}{
					"email":        "JOHN@EXAMPLE.COM",
					"display_name": "JOHNNY",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "all",
	}

	// Should match even though case is different
	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_ExcludesBotsByDefault(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "john-bot",
				"real_name": "John Bot",
				"is_bot":    true,
				"profile": map[string]interface{}{
					"email":        "",
					"display_name": "John Bot",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "all",
	}

	// Should not find the bot
	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_IncludeBots(t *testing.T) {
	response := map[string]interface{}{
		"ok": true,
		"members": []map[string]interface{}{
			{
				"id":        "U123",
				"name":      "john-bot",
				"real_name": "John Bot",
				"is_bot":    true,
				"profile": map[string]interface{}{
					"email":        "",
					"display_name": "John Bot",
				},
			},
		},
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: true,
		field:       "all",
	}

	// Should find the bot when --include-bots is set
	err := runSearch("john", opts, c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunSearchUsers_APIError(t *testing.T) {
	response := map[string]interface{}{
		"ok":    false,
		"error": "not_authed",
	}

	c, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	opts := &searchOptions{
		limit:       100,
		includeBots: false,
		field:       "all",
	}

	err := runSearch("john", opts, c)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestValidateField(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		wantError bool
	}{
		{"all field", "all", false},
		{"name field", "name", false},
		{"email field", "email", false},
		{"display_name field", "display_name", false},
		{"invalid field", "invalid", true},
		{"empty field", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateField(tt.field)
			if (err != nil) != tt.wantError {
				t.Errorf("validateField(%q) error = %v, wantError %v", tt.field, err, tt.wantError)
			}
		})
	}
}

func TestMatchesQuery(t *testing.T) {
	user := client.User{
		ID:       "U123",
		Name:     "john.doe",
		RealName: "John Doe",
		IsBot:    false,
	}
	user.Profile.Email = "john@example.com"
	user.Profile.DisplayName = "Johnny"

	tests := []struct {
		name  string
		query string
		field string
		want  bool
	}{
		{"all field - matches name", "john", "all", true},
		{"all field - matches email", "example.com", "all", true},
		{"all field - matches display", "johnny", "all", true},
		{"all field - no match", "alice", "all", false},
		{"name field - matches", "john.doe", "name", true},
		{"name field - no match (email)", "example.com", "name", false},
		{"email field - matches", "example.com", "email", true},
		{"email field - no match (name)", "john.doe", "email", false},
		{"display_name field - matches display", "johnny", "display_name", true},
		{"display_name field - matches real_name", "doe", "display_name", true},
		{"display_name field - no match", "alice", "display_name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesQuery(user, tt.query, tt.field)
			if got != tt.want {
				t.Errorf("matchesQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}
