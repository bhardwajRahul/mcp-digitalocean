package functions

import (
	"container/list"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestActivationTool_ListActivations(t *testing.T) {
	activations := []map[string]any{
		{"activationId": "act-1", "name": "hello", "start": 1700000000000},
		{"activationId": "act-2", "name": "greet", "start": 1700000001000},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/activations")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activations)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActivationTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"NamespaceID": "ns-uuid-1"}
	resp, err := tool.listActivations(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "act-1")
	require.Contains(t, content, "act-2")
}

func TestActivationTool_ListActivationsWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		q := r.URL.Query()
		require.Equal(t, "hello", q.Get("name"))
		require.Equal(t, "10", q.Get("limit"))
		require.Equal(t, "5", q.Get("skip"))
		require.Equal(t, "1700000000000", q.Get("since"))
		require.Equal(t, "1700000099000", q.Get("upto"))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActivationTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID":  "ns-uuid-1",
		"FunctionName": "hello",
		"Limit":        float64(10),
		"Skip":         float64(5),
		"Since":        float64(1700000000000),
		"Upto":         float64(1700000099000),
	}
	resp, err := tool.listActivations(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestActivationTool_GetActivation(t *testing.T) {
	activation := map[string]any{
		"activationId": "act-123",
		"name":         "hello",
		"namespace":    "test-ns",
		"start":        1700000000000,
		"end":          1700000000100,
		"response": map[string]any{
			"status":  "success",
			"success": true,
			"result":  map[string]any{"body": "Hello!"},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/activations/act-123")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activation)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActivationTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID":  "ns-uuid-1",
		"ActivationID": "act-123",
	}
	resp, err := tool.getActivation(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "act-123")
	require.Contains(t, content, "Hello!")
}

func TestActivationTool_GetActivationLogs(t *testing.T) {
	logs := map[string]any{
		"logs": []string{"2024-01-01T00:00:00Z stdout: Hello"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/activations/act-123/logs")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(logs)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActivationTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID":  "ns-uuid-1",
		"ActivationID": "act-123",
	}
	resp, err := tool.getActivationLogs(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "Hello")
}

func TestActivationTool_GetActivationResult(t *testing.T) {
	result := map[string]any{"body": "Hello, World!"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/activations/act-123/result")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActivationTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID":  "ns-uuid-1",
		"ActivationID": "act-123",
	}
	resp, err := tool.getActivationResult(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "Hello, World!")
}

func TestActivationTool_MissingRequiredArgs(t *testing.T) {
	resolver := &OWResolver{
		items: make(map[string]*list.Element),
		order: list.New(),
	}
	tool := NewActivationTool(resolver)

	tests := []struct {
		name   string
		method func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args   map[string]interface{}
	}{
		{
			name:   "listActivations missing NamespaceID",
			method: tool.listActivations,
			args:   map[string]interface{}{},
		},
		{
			name:   "getActivation missing ActivationID",
			method: tool.getActivation,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
		{
			name:   "getActivationLogs missing ActivationID",
			method: tool.getActivationLogs,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
		{
			name:   "getActivationResult missing ActivationID",
			method: tool.getActivationResult,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			resp, err := tc.method(context.Background(), req)
			require.NoError(t, err)
			require.True(t, resp.IsError)
		})
	}
}
