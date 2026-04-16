//go:build integration

package testing

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenAIListEvaluationMetrics calls genai-list-evaluation-metrics against the live GenAI API.
// Requires DIGITALOCEAN_API_TOKEN with access to GenAI evaluation endpoints (same as other integration tests).
func TestGenAIListEvaluationMetrics(t *testing.T) {
	t.Parallel()

	type metric struct {
		MetricUUID string `json:"metric_uuid"`
		MetricName string `json:"metric_name"`
	}
	type listMetricsResponse struct {
		Metrics []metric `json:"metrics"`
		Count   int      `json:"count"`
	}

	out := callTool[listMetricsResponse](t, "genai-list-evaluation-metrics", map[string]any{})

	require.Greater(t, out.Count, 0, "expected at least one evaluation metric from API")
	require.Len(t, out.Metrics, out.Count)
	require.NotEmpty(t, out.Metrics[0].MetricUUID)
	require.NotEmpty(t, out.Metrics[0].MetricName)
	t.Logf("listed %d evaluation metric(s)", out.Count)
}

// TestGenAIListEvaluationTestCases lists test cases for a workspace when GENAI_EVALUATION_TEST_AGENT_WORKSPACE_NAME is set.
// Skip locally or in CI when the variable is unset; use a real agent workspace name from your account.
func TestGenAIListEvaluationTestCases(t *testing.T) {
	t.Parallel()

	workspace := os.Getenv("GENAI_EVALUATION_TEST_AGENT_WORKSPACE_NAME")
	if workspace == "" {
		t.Skip("set GENAI_EVALUATION_TEST_AGENT_WORKSPACE_NAME to run this test")
	}

	type listTestCasesResponse struct {
		TestCases []struct {
			TestCaseUUID string `json:"test_case_uuid"`
			Name         string `json:"name"`
		} `json:"test_cases"`
		Count int `json:"count"`
	}

	out := callTool[listTestCasesResponse](t, "genai-list-evaluation-test-cases", map[string]any{
		"agent_workspace_name": workspace,
	})

	require.GreaterOrEqual(t, out.Count, 0)
	require.Len(t, out.TestCases, out.Count)
	t.Logf("workspace %q: %d evaluation test case(s)", workspace, out.Count)
}
