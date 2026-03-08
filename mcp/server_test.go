package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/KaioH3/SearchEng/engine"
)

type mockProvider struct {
	name    string
	results []engine.Result
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Search(ctx context.Context, query string, page int) ([]engine.Result, error) {
	return m.results, nil
}

func newTestEngine() *engine.Engine {
	return &engine.Engine{
		Providers: []engine.Provider{
			&mockProvider{
				name: "Mock",
				results: []engine.Result{
					{URL: "https://example.com", Title: "Example", Snippet: "Test snippet about Go programming", Source: "Mock"},
				},
			},
		},
		Timeout:    5 * time.Second,
		MaxResults: 20,
		SafeSearch: true,
	}
}

func TestInitialize(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"capabilities":{}}}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want 2024-11-05", result["protocolVersion"])
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo map")
	}
	if info["name"] != "searcheng" {
		t.Errorf("name = %v, want searcheng", info["name"])
	}
}

func TestToolsList(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"tools/list","id":2}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	json.Unmarshal(out.Bytes(), &resp)

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("expected result map")
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected non-empty tools list")
	}
	tool := tools[0].(map[string]any)
	if tool["name"] != "search" {
		t.Errorf("tool name = %v, want search", tool["name"])
	}
}

func TestToolsCall_Search(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"tools/call","id":3,"params":{"name":"search","arguments":{"query":"Go programming"}}}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	json.Unmarshal(out.Bytes(), &resp)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	content, ok := resp.Result.([]any)
	if !ok || len(content) == 0 {
		t.Fatal("expected content array")
	}
	item := content[0].(map[string]any)
	text, ok := item["text"].(string)
	if !ok || text == "" {
		t.Error("expected non-empty text in response")
	}
	if !strings.Contains(text, "Example") {
		t.Errorf("expected result title in text, got: %s", text)
	}
}

func TestToolsCall_MissingQuery(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"tools/call","id":4,"params":{"name":"search","arguments":{"query":""}}}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	json.Unmarshal(out.Bytes(), &resp)

	if resp.Error == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestToolsCall_UnknownTool(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"tools/call","id":5,"params":{"name":"unknown","arguments":{}}}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	json.Unmarshal(out.Bytes(), &resp)

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestUnknownMethod(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"nonexistent","id":6}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	var resp jsonrpcResponse
	json.Unmarshal(out.Bytes(), &resp)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	if out.Len() != 0 {
		t.Errorf("expected no output for notification, got: %s", out.String())
	}
}

func TestMultipleRequests(t *testing.T) {
	srv := NewServer(newTestEngine())
	input := `{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"capabilities":{}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","method":"tools/list","id":2}
`
	var out bytes.Buffer

	srv.Run(strings.NewReader(input), &out)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// Should have 2 responses (initialize + tools/list), notification has no response
	if len(lines) != 2 {
		t.Errorf("expected 2 response lines, got %d: %v", len(lines), lines)
	}
}
