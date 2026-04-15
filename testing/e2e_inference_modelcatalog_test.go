//go:build integration

package testing

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// TestModelCatalogSearch tests searching for models in the catalog
func TestModelCatalogSearch(t *testing.T) {
	// Search for a common model name
	t.Log("searching for 'llama' models...")
	result := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": "llama",
	})

	require.NotNil(t, result)
	require.Equal(t, "llama", result.SearchQuery)
	t.Logf("found %d models matching 'llama'", result.Count)

	// Search with empty string to get all models
	t.Log("searching with empty string to get all models...")
	allModels := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": "",
	})

	require.NotNil(t, allModels)
	require.Equal(t, "", allModels.SearchQuery)
	require.Greater(t, allModels.Count, 0, "empty search should return all models")
	t.Logf("found %d total models with empty search", allModels.Count)

	// Search with missing parameter (should also return all models)
	t.Log("searching with missing parameter to get all models...")
	allModelsNoParam := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{})

	require.NotNil(t, allModelsNoParam)
	require.Equal(t, "", allModelsNoParam.SearchQuery, "missing parameter should default to empty string")
	require.Greater(t, allModelsNoParam.Count, 0, "missing parameter should return all models")
	require.Equal(t, allModels.Count, allModelsNoParam.Count, "empty string and missing parameter should return same results")
	t.Logf("found %d total models with missing parameter", allModelsNoParam.Count)

	// Search for a model that likely doesn't exist
	t.Log("searching for non-existent model...")
	emptyResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": "nonexistent-model-xyz-123",
	})

	require.NotNil(t, emptyResult)
	require.Equal(t, 0, emptyResult.Count)
	require.Empty(t, emptyResult.ModelUUIDs)
	t.Log("correctly returned empty results for non-existent model")
}

// TestModelCatalogGetCard tests retrieving model metadata
func TestModelCatalogGetCard(t *testing.T) {
	// First search for models to get a valid UUID
	searchQuery := "llama"
	t.Logf("searching for '%s' models to get a valid UUID...", searchQuery)
	searchResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": searchQuery,
	})

	if len(searchResult.ModelUUIDs) == 0 {
		t.Skip("no models found to test with, skipping model card test")
	}

	// Use the first UUID to get model details
	testUUID := searchResult.ModelUUIDs[0]
	t.Logf("getting model card for UUID: %s", testUUID)

	model := callTool[struct {
		UUID              string                `json:"uuid"`
		Name              string                `json:"name"`
		Description       string                `json:"description,omitempty"`
		Provider          string                `json:"provider,omitempty"`
		Agreement         *godo.Agreement       `json:"agreement,omitempty"`
		ModelAvailability string                `json:"model_availability,omitempty"`
		ContextWindow     string                `json:"context_window,omitempty"`
		Capabilities      []string              `json:"capabilities,omitempty"`
		Modalities        *godo.ModelModalities `json:"modalities,omitempty"`
		ParameterCount    float64               `json:"parameter_count,omitempty"`
		Type              string                `json:"type,omitempty"`
		Pricing           *godo.ModelPricing    `json:"pricing,omitempty"`
		BenchmarkScore    json.RawMessage       `json:"benchmark_score,omitempty"`
	}](t, "inference-model-catalog-get-card", map[string]interface{}{
		"ModelUUID": testUUID,
	})

	require.Equal(t, testUUID, model.UUID)
	require.NotEmpty(t, model.Name, "model should have a name")
	require.Contains(t, strings.ToLower(model.Name), strings.ToLower(searchQuery), "model name should contain the search query")
	t.Logf("successfully retrieved model card: %s", model.Name)

	if model.Description != "" {
		t.Logf("description: %s", model.Description)
	}
	if model.Provider != "" {
		t.Logf("provider: %s", model.Provider)
	}
	if model.ModelAvailability != "" {
		t.Logf("model availability: %s", model.ModelAvailability)
	}
	if model.ContextWindow != "" {
		t.Logf("context window: %s", model.ContextWindow)
	}
	if len(model.Capabilities) > 0 {
		t.Logf("capabilities: %v", model.Capabilities)
	}
	if model.ParameterCount > 0 {
		t.Logf("parameter count: %.1fB", model.ParameterCount)
	}
	if model.Type != "" {
		t.Logf("type: %s", model.Type)
	}
	if model.Modalities != nil {
		t.Logf("modalities - input: %v, output: %v", model.Modalities.Input, model.Modalities.Output)
	}
	if model.Pricing != nil {
		t.Logf("pricing - input: $%.2f/M, output: $%.2f/M", model.Pricing.InputPricePerMillion, model.Pricing.OutputPricePerMillion)
	}
	if len(model.BenchmarkScore) > 0 && string(model.BenchmarkScore) != "null" {
		t.Logf("benchmark score: %s", string(model.BenchmarkScore))
	}

	// Verify model metadata structure
	if model.Agreement != nil {
		t.Logf("model agreement: %s", model.Agreement.Name)
	}
}

// TestModelCatalogGetCardNotFound tests error handling for non-existent model UUID
func TestModelCatalogGetCardNotFound(t *testing.T) {
	ctx, c := getTestClient(t)

	fakeUUID := "99999999-9999-9999-9999-999999999999"
	t.Logf("attempting to get model card for non-existent UUID: %s", fakeUUID)

	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "inference-model-catalog-get-card",
			Arguments: map[string]string{
				"ModelUUID": fakeUUID,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.IsError, "should return error for non-existent UUID")

	if len(resp.Content) > 0 {
		if tc, ok := resp.Content[0].(mcp.TextContent); ok {
			t.Logf("error message: %s", tc.Text)
			require.Contains(t, tc.Text, "not found")
		}
	}
}

// TestModelCatalogWorkflow tests a complete workflow: search -> get details
func TestModelCatalogWorkflow(t *testing.T) {
	t.Log("testing complete model catalog workflow...")

	// Step 1: Search for a specific model
	searchQuery := "gpt"
	t.Logf("step 1: searching for '%s' models...", searchQuery)

	searchResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": searchQuery,
	})

	t.Logf("found %d models matching '%s'", searchResult.Count, searchQuery)

	if len(searchResult.ModelUUIDs) == 0 {
		t.Skip("no models found, skipping workflow test")
	}

	// Step 2: Get details for each model found
	t.Logf("step 2: getting details for %d models...", len(searchResult.ModelUUIDs))

	for i, uuid := range searchResult.ModelUUIDs {
		if i >= 3 {
			// Limit to first 3 models to avoid too many API calls
			t.Logf("limiting to first 3 models")
			break
		}

		model := callTool[struct {
			UUID              string                `json:"uuid"`
			Name              string                `json:"name"`
			Description       string                `json:"description,omitempty"`
			Provider          string                `json:"provider,omitempty"`
			Agreement         *godo.Agreement       `json:"agreement,omitempty"`
			ModelAvailability string                `json:"model_availability,omitempty"`
			ContextWindow     string                `json:"context_window,omitempty"`
			Capabilities      []string              `json:"capabilities,omitempty"`
			Modalities        *godo.ModelModalities `json:"modalities,omitempty"`
			ParameterCount    float64               `json:"parameter_count,omitempty"`
			Type              string                `json:"type,omitempty"`
			Pricing           *godo.ModelPricing    `json:"pricing,omitempty"`
			BenchmarkScore    json.RawMessage       `json:"benchmark_score,omitempty"`
		}](t, "inference-model-catalog-get-card", map[string]interface{}{
			"ModelUUID": uuid,
		})

		require.Equal(t, uuid, model.UUID)
		require.Contains(t, strings.ToLower(model.Name), strings.ToLower(searchQuery), "model name should contain the search query")
		if model.ModelAvailability != "" {
			t.Logf("  - %s (availability: %s)", model.Name, model.ModelAvailability)
		} else {
			t.Logf("  - %s", model.Name)
		}
		if model.Provider != "" {
			t.Logf("    provider: %s", model.Provider)
		}
		if model.ContextWindow != "" {
			t.Logf("    context window: %s", model.ContextWindow)
		}
	}

	t.Log("workflow completed successfully")
}

// TestModelComparisonPrompt tests the model comparison prompt
func TestModelComparisonPrompt(t *testing.T) {
	ctx, c := getTestClient(t)

	// First, get two model UUIDs to compare
	t.Log("searching for models to compare...")
	searchResult := callTool[struct {
		ModelUUIDs  []string `json:"model_uuids"`
		SearchQuery string   `json:"search_query"`
		Count       int      `json:"count"`
	}](t, "inference-model-catalog-search", map[string]interface{}{
		"SearchQuery": "gpt",
	})

	if len(searchResult.ModelUUIDs) < 2 {
		t.Skip("need at least 2 models to test comparison, skipping")
	}

	uuid1 := searchResult.ModelUUIDs[0]
	uuid2 := searchResult.ModelUUIDs[1]
	t.Logf("comparing models: %s vs %s", uuid1, uuid2)

	resp, err := c.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "model-comparison",
			Arguments: map[string]string{
				"ModelUUID1": uuid1,
				"ModelUUID2": uuid2,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Description, "comparison should have a description")
	require.NotEmpty(t, resp.Messages, "comparison should have messages")
	require.Greater(t, len(resp.Messages), 0, "should have at least one message")

	if len(resp.Messages) > 0 {
		msg := resp.Messages[0]
		require.Equal(t, "user", string(msg.Role), "message role should be 'user'")

		if textContent, ok := msg.Content.(mcp.TextContent); ok {
			require.NotEmpty(t, textContent.Text, "message should have text content")
			require.Contains(t, textContent.Text, "Model Comparison", "should contain comparison header")
			require.Contains(t, textContent.Text, "Input Price", "should contain input price")
			require.Contains(t, textContent.Text, "Output Price", "should contain output price")
			require.Contains(t, textContent.Text, "Capabilities", "should contain capabilities section")
			t.Logf("comparison generated successfully with %d characters", len(textContent.Text))
		} else {
			t.Fatalf("expected TextContent, got %T", msg.Content)
		}
	}
}

// TestSearchByTaskPrompt tests the search by task prompt
func TestSearchByTaskPrompt(t *testing.T) {
	ctx, c := getTestClient(t)

	t.Log("searching for models by task with provider constraint to limit results")

	resp, err := c.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "search-by-task",
			Arguments: map[string]string{
				"Task":     "chat",
				"Provider": "anthropic", // Add provider to limit results
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Description, "search should have a description")
	require.Contains(t, resp.Description, "chat", "description should mention the task")
	require.NotEmpty(t, resp.Messages, "search should have messages")
	require.Greater(t, len(resp.Messages), 0, "should have at least one message")

	if len(resp.Messages) > 0 {
		msg := resp.Messages[0]
		require.Equal(t, "user", string(msg.Role), "message role should be 'user'")

		if textContent, ok := msg.Content.(mcp.TextContent); ok {
			require.NotEmpty(t, textContent.Text, "message should have text content")
			require.Contains(t, textContent.Text, "Model Search Results", "should contain search header")
			require.Contains(t, textContent.Text, "chat", "should mention the task")
			require.Contains(t, textContent.Text, "Input Price", "should show pricing information")
			t.Logf("search results generated successfully with %d characters", len(textContent.Text))
		} else {
			t.Fatalf("expected TextContent, got %T", msg.Content)
		}
	}
}

// TestSearchByTaskPromptWithConstraints tests search with filtering constraints
func TestSearchByTaskPromptWithConstraints(t *testing.T) {
	ctx, c := getTestClient(t)

	t.Log("searching for models by task with constraints")

	resp, err := c.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "search-by-task",
			Arguments: map[string]string{
				"Task":           "reasoning",
				"DeploymentType": "Serverless",
				"Provider":       "Anthropic",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Description, "search should have a description")
	require.NotEmpty(t, resp.Messages, "search should have messages")

	if len(resp.Messages) > 0 {
		msg := resp.Messages[0]
		require.Equal(t, "user", string(msg.Role), "message role should be 'user'")

		if textContent, ok := msg.Content.(mcp.TextContent); ok {
			require.NotEmpty(t, textContent.Text, "message should have text content")
			require.Contains(t, textContent.Text, "Applied Constraints", "should show constraints section")
			require.Contains(t, textContent.Text, "Serverless", "should mention deployment type constraint")
			require.Contains(t, textContent.Text, "Anthropic", "should mention provider constraint")
			t.Logf("constrained search results generated successfully")
		} else {
			t.Fatalf("expected TextContent, got %T", msg.Content)
		}
	}
}

// TestModelComparisonPromptInvalidUUID tests error handling for comparison with invalid UUID
func TestModelComparisonPromptInvalidUUID(t *testing.T) {
	ctx, c := getTestClient(t)

	t.Log("testing comparison with invalid UUID")

	_, err := c.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "model-comparison",
			Arguments: map[string]string{
				"ModelUUID1": "99999999-9999-9999-9999-999999999999",
				"ModelUUID2": "88888888-8888-8888-8888-888888888888",
			},
		},
	})

	require.Error(t, err, "should return error for invalid UUIDs")
	t.Logf("correctly returned error: %v", err)
}

// TestSearchByTaskPromptNoModelsFound tests search with impossible constraints (UAT 3.4.9)
func TestSearchByTaskPromptNoModelsFound(t *testing.T) {
	ctx, c := getTestClient(t)

	t.Log("testing search by task with impossible constraints")

	resp, err := c.GetPrompt(ctx, mcp.GetPromptRequest{
		Params: mcp.GetPromptParams{
			Name: "search-by-task",
			Arguments: map[string]string{
				"Task":             "chat",
				"Provider":         "NonExistentProvider",
				"MinContextWindow": "999999999",
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp.Messages, "search should have messages")

	if len(resp.Messages) > 0 {
		msg := resp.Messages[0]
		require.Equal(t, "user", string(msg.Role), "message role should be 'user'")

		if textContent, ok := msg.Content.(mcp.TextContent); ok {
			require.NotEmpty(t, textContent.Text, "message should have text content")
			require.Contains(t, textContent.Text, "Applied Constraints", "should show constraints section")
			require.Contains(t, textContent.Text, "NonExistentProvider", "should show provider constraint")
			require.Contains(t, textContent.Text, "No Models Found", "should state no models found")
			require.NotContains(t, textContent.Text, "Recommendation", "should not have recommendation section")
			t.Logf("correctly returned empty result with honest message")
		} else {
			t.Fatalf("expected TextContent, got %T", msg.Content)
		}
	}
}
