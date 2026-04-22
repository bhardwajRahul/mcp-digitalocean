package functions

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestDeploymentGuideTool_ReturnsEmbeddedSpec(t *testing.T) {
	tool := NewDeploymentGuideTool()

	resp, err := tool.getDeploymentGuide(context.Background(), mcp.CallToolRequest{})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.NotEmpty(t, resp.Content)

	text := resp.Content[0].(mcp.TextContent).Text
	require.NotEmpty(t, text, "deployment guide must not be empty")

	// Sanity-check that the embedded content is the deploy spec, not some
	// other file. These headings must stay stable; if they change, update
	// the assertions along with the spec.
	for _, marker := range []string{
		"DigitalOcean Functions",
		"doctl serverless",
		"Preflight",
		"Deploy",
	} {
		require.True(t,
			strings.Contains(text, marker),
			"deployment guide missing expected marker %q", marker,
		)
	}
}

func TestDeploymentGuideTool_ToolMetadata(t *testing.T) {
	tool := NewDeploymentGuideTool()
	tools := tool.Tools()
	require.Len(t, tools, 1)

	st := tools[0]
	require.Equal(t, "functions-deployment-guide", st.Tool.Name)
	require.NotEmpty(t, st.Tool.Description)
	require.NotNil(t, st.Handler)
}
