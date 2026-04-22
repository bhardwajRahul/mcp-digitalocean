//go:build integration

package testing

import (
	"os"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// TestGenAIBatchInferenceListJobs lists batch inference jobs against the live GenAI API.
// Requires DIGITALOCEAN_API_TOKEN with access to batch inference endpoints and the feature enabled.
func TestGenAIBatchInferenceListJobs(t *testing.T) {
	t.Parallel()

	ctx, c := getTestClient(t)
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-batch-inference-list",
			Arguments: map[string]any{},
		},
	})
	require.NoError(t, err)

	if resp.IsError {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			if strings.Contains(tc.Text, "not enabled") ||
				strings.Contains(tc.Text, "permission") ||
				strings.Contains(tc.Text, "403") {
				t.Skip("batch inference not enabled for test account")
			}
		}
		t.Fatalf("list tool failed: %v", resp.Content)
	}

	list := callTool[godo.ListBatchesResponse](t, "genai-batch-inference-list", map[string]any{})
	t.Logf("listed %d batch inference job edge(s), hasNextPage=%v", len(list.Edges), list.PageInfo.HasNextPage)
}

// TestGenAIBatchInferenceFileUploadAndCreate tests the file upload + job create flow.
// Set GENAI_BATCH_INFERENCE_E2E=1 to run (requires feature-enabled account).
func TestGenAIBatchInferenceFileUploadAndCreate(t *testing.T) {
	t.Parallel()

	if os.Getenv("GENAI_BATCH_INFERENCE_E2E") == "" {
		t.Skip("set GENAI_BATCH_INFERENCE_E2E=1 to run batch inference E2E tests")
	}

	upload := callTool[godo.CreateBatchFileResponse](t, "genai-batch-inference-create-file", map[string]any{
		"FileName": "e2e-test-input.jsonl",
	})

	require.NotEmpty(t, upload.FileID, "expected non-empty file_id")
	require.NotEmpty(t, upload.UploadURL, "expected non-empty upload_url")
	t.Logf("created file upload: file_id=%s", upload.FileID)

	batch := callTool[godo.Batch](t, "genai-batch-inference-create", map[string]any{
		"Provider":         "openai",
		"CompletionWindow": "24h",
		"FileID":           upload.FileID,
		"Endpoint":         "/v1/chat/completions",
	})

	require.NotEmpty(t, batch.BatchID, "expected non-empty batch_id")
	require.Equal(t, "openai", batch.Provider)
	t.Logf("created batch job: batch_id=%s, status=%s", batch.BatchID, batch.Status)

	poll := callTool[godo.Batch](t, "genai-batch-inference-get", map[string]any{
		"BatchID": batch.BatchID,
	})

	require.Equal(t, batch.BatchID, poll.BatchID)
	t.Logf("polled batch job: status=%s", poll.Status)

	list := callTool[godo.ListBatchesResponse](t, "genai-batch-inference-list", map[string]any{})

	found := false
	for _, edge := range list.Edges {
		if edge.Node.BatchID == batch.BatchID {
			found = true
			break
		}
	}
	require.True(t, found, "created job should appear in list")

	t.Logf("attempting best-effort cancel of batch job %s", batch.BatchID)
	ctx, c := getTestClient(t)
	_, _ = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "genai-batch-inference-cancel",
			Arguments: map[string]any{"BatchID": batch.BatchID},
		},
	})
}
