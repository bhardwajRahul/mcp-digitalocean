package functions

import (
	"container/list"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// newTestResolver creates an OWResolver pre-seeded with a cached auth entry
// pointing at the given test server. This lets us test tool handlers without
// needing a real godo client.
func newTestResolver(t *testing.T, ts *httptest.Server, nsID, nsName string) *OWResolver {
	t.Helper()
	ow := newOWClient(ts.URL, "test-key-id:test-secret")
	resolver := &OWResolver{
		client: nil,
		items:  make(map[string]*list.Element),
		order:  list.New(),
	}
	entry := &lruEntry{
		key: nsID,
		auth: &cachedAuth{
			ow:         ow,
			nsName:     nsName,
			validUntil: time.Now().Add(1 * time.Hour),
		},
	}
	elem := resolver.order.PushFront(entry)
	resolver.items[nsID] = elem
	return resolver
}

func TestActionTool_ListActions(t *testing.T) {
	actions := []map[string]any{
		{"name": "hello", "namespace": "test-ns", "version": "0.0.1"},
		{"name": "greet", "namespace": "test-ns", "version": "0.0.2"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(actions)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"NamespaceID": "ns-uuid-1"}
	resp, err := tool.listActions(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "hello")
	require.Contains(t, content, "greet")
}

func TestActionTool_GetAction(t *testing.T) {
	action := map[string]any{
		"name":      "hello",
		"namespace": "test-ns",
		"version":   "0.0.1",
		"exec":      map[string]any{"kind": "nodejs:22"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions/hello")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(action)
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "hello",
	}
	resp, err := tool.getAction(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "hello")
	require.Contains(t, content, "nodejs:22")
}

func TestActionTool_GetAction_WithPackage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions/mypkg/hello")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"name": "hello"})
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "hello",
		"PackageName": "mypkg",
	}
	resp, err := tool.getAction(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
}

func TestActionTool_CreateOrUpdateAction(t *testing.T) {
	var receivedBody map[string]any

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPut, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions/hello")
		require.Equal(t, "true", r.URL.Query().Get("overwrite"))

		json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"name":      "hello",
			"namespace": "test-ns",
			"version":   "0.0.1",
		})
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "hello",
		"Kind":        "nodejs:22",
		"Code":        `function main() { return { body: "hi" } }`,
		"Timeout":     float64(30000),
		"Memory":      float64(512),
	}
	resp, err := tool.createOrUpdateAction(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)

	exec, _ := receivedBody["exec"].(map[string]any)
	require.Equal(t, "nodejs:22", exec["kind"])
	require.NotEmpty(t, exec["code"])

	limits, _ := receivedBody["limits"].(map[string]any)
	require.Equal(t, float64(30000), limits["timeout"])
	require.Equal(t, float64(512), limits["memory"])
}

func TestActionTool_DeleteAction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodDelete, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions/hello")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "hello",
	}
	resp, err := tool.deleteAction(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "deleted successfully")
}

func TestActionTool_InvokeAction(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Contains(t, r.URL.Path, "/namespaces/test-ns/actions/hello")
		require.Equal(t, "true", r.URL.Query().Get("blocking"))
		require.Equal(t, "false", r.URL.Query().Get("result"))

		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		require.Equal(t, "MCP", payload["name"])

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"activationId": "act-123",
			"response": map[string]any{
				"result": map[string]any{"body": "Hello, MCP!"},
			},
		})
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "hello",
		"Payload":     map[string]interface{}{"name": "MCP"},
		"Blocking":    true,
	}
	resp, err := tool.invokeAction(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "act-123")
	require.Contains(t, content, "Hello, MCP!")
}

func TestActionTool_MissingRequiredArgs(t *testing.T) {
	resolver := &OWResolver{
		items: make(map[string]*list.Element),
		order: list.New(),
	}
	tool := NewActionTool(resolver)

	tests := []struct {
		name   string
		method func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
		args   map[string]interface{}
	}{
		{
			name:   "listActions missing NamespaceID",
			method: tool.listActions,
			args:   map[string]interface{}{},
		},
		{
			name:   "getAction missing ActionName",
			method: tool.getAction,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
		{
			name:   "createOrUpdateAction missing ActionName",
			method: tool.createOrUpdateAction,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
		{
			name:   "deleteAction missing ActionName",
			method: tool.deleteAction,
			args:   map[string]interface{}{"NamespaceID": "ns-1"},
		},
		{
			name:   "invokeAction missing ActionName",
			method: tool.invokeAction,
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

func TestActionTool_OWAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "The requested resource does not exist."})
	}))
	defer ts.Close()

	resolver := newTestResolver(t, ts, "ns-uuid-1", "test-ns")
	tool := NewActionTool(resolver)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"ActionName":  "nonexistent",
	}
	resp, err := tool.getAction(context.Background(), req)
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestActionPath(t *testing.T) {
	require.Equal(t, "/namespaces/ns/actions/hello", actionPath("ns", "", "hello"))
	require.Equal(t, "/namespaces/ns/actions/pkg/hello", actionPath("ns", "pkg", "hello"))
}
