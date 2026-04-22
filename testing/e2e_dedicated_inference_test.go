//go:build integration

package testing

import (
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

// TestDedicatedInferenceList exercises the dedicated-inference MCP tools against the live API (list only; no create/delete).
func TestDedicatedInferenceList(t *testing.T) {
	t.Parallel()

	out := callTool[struct {
		Items []godo.DedicatedInferenceListItem `json:"items"`
		Meta  *godo.Meta                        `json:"meta,omitempty"`
	}](t, "dedicated-inference-list", map[string]any{})

	require.NotNil(t, out.Items)
	t.Logf("listed %d dedicated inference instance(s)", len(out.Items))
}
