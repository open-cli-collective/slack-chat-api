package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsChannelID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"C123ABC", true},
		{"C02DF3BEUGN", true},
		{"G123456", true},
		{"D123456", true},
		{"CAAAAAAAA", false}, // pure letters - treated as name
		{"GENERAL", false},   // pure letters - treated as name
		{"general", false},
		{"engineering-team", false},
		{"#general", false},
		{"c123abc", false}, // lowercase
		{"X123456", false}, // wrong prefix
		{"C", false},       // too short
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsChannelID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveChannel_ID(t *testing.T) {
	c := NewWithConfig("http://localhost", "test-token", nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"C02DF3BEUGN", "C02DF3BEUGN"},
		{"G123456", "G123456"},
		{"D123456", "D123456"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := c.ResolveChannel(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveChannel_Name(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/conversations.list", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C111111", "name": "general"},
				{"id": "C222222", "name": "engineering-team"},
				{"id": "C333333", "name": "random"},
			},
		})
	}))
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)

	tests := []struct {
		input    string
		expected string
	}{
		{"general", "C111111"},
		{"engineering-team", "C222222"},
		{"#general", "C111111"},
		{"GENERAL", "C111111"}, // case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := c.ResolveChannel(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveChannel_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":       true,
			"channels": []map[string]interface{}{},
		})
	}))
	defer server.Close()

	c := NewWithConfig(server.URL, "test-token", nil)

	_, err := c.ResolveChannel("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "slck channels list")
}

func TestResolveChannel_Empty(t *testing.T) {
	c := NewWithConfig("http://localhost", "test-token", nil)

	_, err := c.ResolveChannel("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}
