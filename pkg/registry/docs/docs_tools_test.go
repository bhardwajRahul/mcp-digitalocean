package docs

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestSearchDocs_MissingQuery(t *testing.T) {
	tool := NewDocsTool()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}

	resp, err := tool.searchDocs(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "Query is required")
}

func TestGetDoc_MissingURL(t *testing.T) {
	tool := NewDocsTool()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}

	resp, err := tool.getDoc(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "URL is required")
}

func TestFindDocsForService_MissingService(t *testing.T) {
	tool := NewDocsTool()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}

	resp, err := tool.findDocsForService(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "Service is required")
}

func TestGetQuickstart_MissingService(t *testing.T) {
	tool := NewDocsTool()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}

	resp, err := tool.getQuickstart(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "Service is required")
}

func TestDocsTool_Tools(t *testing.T) {
	tool := NewDocsTool()
	tools := tool.Tools()

	require.Len(t, tools, 4)

	toolNames := make([]string, len(tools))
	for i, st := range tools {
		toolNames[i] = st.Tool.Name
	}

	require.Contains(t, toolNames, "docs-search")
	require.Contains(t, toolNames, "docs-get-page")
	require.Contains(t, toolNames, "docs-find-for-service")
	require.Contains(t, toolNames, "docs-get-quickstart")

	// Verify all tools have handlers
	for _, st := range tools {
		require.NotNil(t, st.Handler, "tool %s should have a handler", st.Tool.Name)
	}
}
