package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewWithConfig(t *testing.T) {
	client := NewWithConfig("https://example.com", "test-token", nil)

	if client.baseURL != "https://example.com" {
		t.Errorf("expected baseURL to be https://example.com, got %s", client.baseURL)
	}
	if client.token != "test-token" {
		t.Errorf("expected token to be test-token, got %s", client.token)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestNewWithConfig_CustomHTTPClient(t *testing.T) {
	customClient := &http.Client{}
	client := NewWithConfig("https://example.com", "test-token", customClient)

	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be used")
	}
}

func TestClient_GetChannelInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "conversations.info") {
			t.Errorf("expected path to contain conversations.info, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("channel") != "C123456" {
			t.Errorf("expected channel=C123456, got %s", r.URL.Query().Get("channel"))
		}

		// Verify auth header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("expected Authorization header with Bearer prefix")
		}

		// Return success response
		resp := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":          "C123456",
				"name":        "general",
				"is_private":  false,
				"is_archived": false,
				"num_members": 42,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channel, err := client.GetChannelInfo("C123456")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel.ID != "C123456" {
		t.Errorf("expected channel ID C123456, got %s", channel.ID)
	}
	if channel.Name != "general" {
		t.Errorf("expected channel name general, got %s", channel.Name)
	}
	if channel.NumMembers != 42 {
		t.Errorf("expected 42 members, got %d", channel.NumMembers)
	}
}

func TestClient_GetChannelInfo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok":    false,
			"error": "channel_not_found",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	_, err := client.GetChannelInfo("C999999")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "channel_not_found") {
		t.Errorf("expected error to contain channel_not_found, got %s", err.Error())
	}
}

func TestClient_ListChannels_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}

		resp := map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C1", "name": "general", "num_members": 10},
				{"id": "C2", "name": "random", "num_members": 5},
			},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channels, err := client.ListChannels("public_channel", true, 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(channels))
	}
	if channels[0].Name != "general" {
		t.Errorf("expected first channel to be general, got %s", channels[0].Name)
	}
}

func TestClient_ListChannels_Pagination(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp map[string]interface{}

		if callCount == 1 {
			// First page
			resp = map[string]interface{}{
				"ok": true,
				"channels": []map[string]interface{}{
					{"id": "C1", "name": "general"},
				},
				"response_metadata": map[string]string{
					"next_cursor": "cursor123",
				},
			}
		} else {
			// Verify cursor was passed
			if r.URL.Query().Get("cursor") != "cursor123" {
				t.Errorf("expected cursor=cursor123, got %s", r.URL.Query().Get("cursor"))
			}
			// Second page (last)
			resp = map[string]interface{}{
				"ok": true,
				"channels": []map[string]interface{}{
					{"id": "C2", "name": "random"},
				},
				"response_metadata": map[string]string{
					"next_cursor": "",
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channels, err := client.ListChannels("", true, 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", callCount)
	}
	if len(channels) != 2 {
		t.Errorf("expected 2 channels total, got %d", len(channels))
	}
}

func TestClient_GetUserInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("user") != "U123456" {
			t.Errorf("expected user=U123456, got %s", r.URL.Query().Get("user"))
		}

		resp := map[string]interface{}{
			"ok": true,
			"user": map[string]interface{}{
				"id":        "U123456",
				"name":      "testuser",
				"real_name": "Test User",
				"is_admin":  true,
				"is_bot":    false,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	user, err := client.GetUserInfo("U123456")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "U123456" {
		t.Errorf("expected user ID U123456, got %s", user.ID)
	}
	if user.RealName != "Test User" {
		t.Errorf("expected real name Test User, got %s", user.RealName)
	}
	if !user.IsAdmin {
		t.Error("expected user to be admin")
	}
}

func TestClient_ListUsers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{"id": "U1", "name": "user1"},
				{"id": "U2", "name": "user2"},
			},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	users, err := client.ListUsers(100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestClient_SendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "chat.postMessage") {
			t.Errorf("expected path to contain chat.postMessage, got %s", r.URL.Path)
		}

		// Verify request body
		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["channel"] != "C123" {
			t.Errorf("expected channel C123, got %v", reqBody["channel"])
		}
		if reqBody["text"] != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got %v", reqBody["text"])
		}

		resp := map[string]interface{}{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C123",
			"message": map[string]interface{}{
				"text": "Hello, World!",
				"user": "U123",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	msg, err := client.SendMessage("C123", "Hello, World!", "", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.TS != "1234567890.123456" {
		t.Errorf("expected ts 1234567890.123456, got %s", msg.TS)
	}
}

func TestClient_SendMessage_WithThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		if reqBody["thread_ts"] != "1111111111.111111" {
			t.Errorf("expected thread_ts, got %v", reqBody["thread_ts"])
		}

		resp := map[string]interface{}{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C123",
			"message": map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	_, err := client.SendMessage("C123", "Reply", "1111111111.111111", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_SendMessage_WithBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		blocks, ok := reqBody["blocks"].([]interface{})
		if !ok || len(blocks) != 1 {
			t.Errorf("expected 1 block, got %v", reqBody["blocks"])
		}

		resp := map[string]interface{}{
			"ok":      true,
			"ts":      "1234567890.123456",
			"channel": "C123",
			"message": map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	blocks := []interface{}{
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": "Hello",
			},
		},
	}

	client := NewWithConfig(server.URL, "test-token", nil)
	_, err := client.SendMessage("C123", "Hello", "", blocks)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_UpdateMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "chat.update") {
			t.Errorf("expected path to contain chat.update, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["ts"] != "1234567890.123456" {
			t.Errorf("expected ts in request body")
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.UpdateMessage("C123", "1234567890.123456", "Updated text", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DeleteMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "chat.delete") {
			t.Errorf("expected path to contain chat.delete, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.DeleteMessage("C123", "1234567890.123456")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetChannelHistory_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.history") {
			t.Errorf("expected path to contain conversations.history, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"ts": "1234567890.123456", "text": "Hello", "user": "U123"},
				{"ts": "1234567890.123457", "text": "World", "user": "U456"},
			},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	messages, err := client.GetChannelHistory("C123", 20, "", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestClient_GetThreadReplies_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.replies") {
			t.Errorf("expected path to contain conversations.replies, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("ts") != "1234567890.123456" {
			t.Errorf("expected ts query param")
		}

		resp := map[string]interface{}{
			"ok": true,
			"messages": []map[string]interface{}{
				{"ts": "1234567890.123456", "text": "Original"},
				{"ts": "1234567890.123457", "text": "Reply 1"},
			},
			"response_metadata": map[string]string{
				"next_cursor": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	messages, err := client.GetThreadReplies("C123", "1234567890.123456", 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

func TestClient_AddReaction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "reactions.add") {
			t.Errorf("expected path to contain reactions.add, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["name"] != "thumbsup" {
			t.Errorf("expected emoji name thumbsup, got %v", reqBody["name"])
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.AddReaction("C123", "1234567890.123456", "thumbsup")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_RemoveReaction_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "reactions.remove") {
			t.Errorf("expected path to contain reactions.remove, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.RemoveReaction("C123", "1234567890.123456", "thumbsup")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_GetTeamInfo_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "team.info") {
			t.Errorf("expected path to contain team.info, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{
			"ok": true,
			"team": map[string]interface{}{
				"id":     "T123456",
				"name":   "Test Workspace",
				"domain": "test-workspace",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	team, err := client.GetTeamInfo()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if team.ID != "T123456" {
		t.Errorf("expected team ID T123456, got %s", team.ID)
	}
	if team.Name != "Test Workspace" {
		t.Errorf("expected team name Test Workspace, got %s", team.Name)
	}
}

func TestClient_CreateChannel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.create") {
			t.Errorf("expected path to contain conversations.create, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["name"] != "new-channel" {
			t.Errorf("expected name new-channel, got %v", reqBody["name"])
		}
		if reqBody["is_private"] != true {
			t.Errorf("expected is_private true, got %v", reqBody["is_private"])
		}

		resp := map[string]interface{}{
			"ok": true,
			"channel": map[string]interface{}{
				"id":         "C999",
				"name":       "new-channel",
				"is_private": true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channel, err := client.CreateChannel("new-channel", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if channel.ID != "C999" {
		t.Errorf("expected channel ID C999, got %s", channel.ID)
	}
	if !channel.IsPrivate {
		t.Error("expected channel to be private")
	}
}

func TestClient_ArchiveChannel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.archive") {
			t.Errorf("expected path to contain conversations.archive, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.ArchiveChannel("C123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_UnarchiveChannel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.unarchive") {
			t.Errorf("expected path to contain conversations.unarchive, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.UnarchiveChannel("C123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_SetChannelTopic_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.setTopic") {
			t.Errorf("expected path to contain conversations.setTopic, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["topic"] != "New Topic" {
			t.Errorf("expected topic 'New Topic', got %v", reqBody["topic"])
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.SetChannelTopic("C123", "New Topic")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_SetChannelPurpose_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.setPurpose") {
			t.Errorf("expected path to contain conversations.setPurpose, got %s", r.URL.Path)
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.SetChannelPurpose("C123", "New Purpose")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_InviteToChannel_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "conversations.invite") {
			t.Errorf("expected path to contain conversations.invite, got %s", r.URL.Path)
		}

		var reqBody map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["users"] != "U1,U2,U3" {
			t.Errorf("expected users 'U1,U2,U3', got %v", reqBody["users"])
		}

		resp := map[string]interface{}{"ok": true}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	err := client.InviteToChannel("C123", []string{"U1", "U2", "U3"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_NetworkError(t *testing.T) {
	// Use a server that immediately closes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close connection without responding
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	_, err := client.GetTeamInfo()

	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}

func TestClient_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	_, err := client.GetTeamInfo()

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestClient_AuthHeader(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		resp := map[string]interface{}{
			"ok":   true,
			"team": map[string]interface{}{"id": "T1", "name": "Test", "domain": "test"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "xoxb-test-token-12345", nil)
	_, _ = client.GetTeamInfo()

	if capturedAuth != "Bearer xoxb-test-token-12345" {
		t.Errorf("expected 'Bearer xoxb-test-token-12345', got '%s'", capturedAuth)
	}
}

func TestClient_ListChannels_LimitTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server returns 5 channels, but we'll request limit=3
		resp := map[string]interface{}{
			"ok": true,
			"channels": []map[string]interface{}{
				{"id": "C1", "name": "ch1"},
				{"id": "C2", "name": "ch2"},
				{"id": "C3", "name": "ch3"},
				{"id": "C4", "name": "ch4"},
				{"id": "C5", "name": "ch5"},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channels, err := client.ListChannels("", true, 3) // Request only 3

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(channels) != 3 {
		t.Errorf("expected 3 channels (limit), got %d", len(channels))
	}
}

func TestClient_ListChannels_PaginationWithLimit(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp map[string]interface{}

		if callCount == 1 {
			// First page: 3 channels, more available
			resp = map[string]interface{}{
				"ok": true,
				"channels": []map[string]interface{}{
					{"id": "C1", "name": "ch1"},
					{"id": "C2", "name": "ch2"},
					{"id": "C3", "name": "ch3"},
				},
				"response_metadata": map[string]string{"next_cursor": "cursor123"},
			}
		} else {
			// Should NOT reach here with limit=3
			t.Error("should not fetch second page when limit already reached")
			resp = map[string]interface{}{
				"ok":       true,
				"channels": []map[string]interface{}{{"id": "C4", "name": "ch4"}},
				"response_metadata": map[string]string{"next_cursor": ""},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	channels, err := client.ListChannels("", true, 3)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (limit reached), got %d", callCount)
	}
	if len(channels) != 3 {
		t.Errorf("expected 3 channels, got %d", len(channels))
	}
}

func TestClient_ListUsers_LimitTruncation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ok": true,
			"members": []map[string]interface{}{
				{"id": "U1", "name": "user1"},
				{"id": "U2", "name": "user2"},
				{"id": "U3", "name": "user3"},
				{"id": "U4", "name": "user4"},
			},
			"response_metadata": map[string]string{"next_cursor": ""},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	users, err := client.ListUsers(2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users (limit), got %d", len(users))
	}
}

func TestClient_ListUsers_PaginationWithLimit(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp map[string]interface{}

		if callCount == 1 {
			resp = map[string]interface{}{
				"ok": true,
				"members": []map[string]interface{}{
					{"id": "U1", "name": "user1"},
					{"id": "U2", "name": "user2"},
				},
				"response_metadata": map[string]string{"next_cursor": "cursor456"},
			}
		} else {
			t.Error("should not fetch second page when limit already reached")
			resp = map[string]interface{}{
				"ok":      true,
				"members": []map[string]interface{}{{"id": "U3", "name": "user3"}},
				"response_metadata": map[string]string{"next_cursor": ""},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewWithConfig(server.URL, "test-token", nil)
	users, err := client.ListUsers(2)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 API call (limit reached), got %d", callCount)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}
