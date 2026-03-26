//go:build integration

package testing

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

// deleteNamespace is a cleanup helper that logs errors but doesn't fail the test.
func deleteNamespace(t *testing.T, tc testContext, namespaceID string) {
	t.Helper()
	t.Logf("deleting serverless namespace %s...", namespaceID)
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "serverless-namespace-delete",
			Arguments: map[string]interface{}{"NamespaceID": namespaceID},
		},
	})
	if err != nil {
		t.Logf("failed to delete namespace: %v", err)
		return
	}
	if resp.IsError {
		t.Logf("serverless-namespace-delete returned error: %v", resp.Content)
		return
	}
	t.Logf("deleted namespace %s", namespaceID)
}

// TestServerlessNamespaceLifecycle tests the full lifecycle of a serverless namespace:
// create -> get -> list -> delete
func TestServerlessNamespaceLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	namespaceLabel := fmt.Sprintf("mcp-e2e-ns-%d", time.Now().Unix())

	// create namespace
	t.Log("creating serverless namespace...")
	namespace := callTool[godo.FunctionsNamespace](t, "serverless-namespace-create", map[string]interface{}{
		"Label":  namespaceLabel,
		"Region": testRegionNYC1,
	})
	require.NotEmpty(t, namespace.Namespace, "namespace ID should not be empty")
	require.Equal(t, namespaceLabel, namespace.Label)
	t.Logf("created namespace: %s (ID: %s, region: %s)", namespace.Label, namespace.Namespace, namespace.Region)

	defer func() { deleteNamespace(t, tc, namespace.Namespace) }()

	// get namespace
	t.Log("getting serverless namespace...")
	fetched := callTool[godo.FunctionsNamespace](t, "serverless-namespace-get", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
	})
	require.Equal(t, namespace.Namespace, fetched.Namespace)
	require.Equal(t, namespaceLabel, fetched.Label)
	t.Logf("fetched namespace: %s (label: %s)", fetched.Namespace, fetched.Label)

	// list namespaces
	t.Log("listing serverless namespaces...")
	namespaces := callTool[[]godo.FunctionsNamespace](t, "serverless-namespace-list", map[string]interface{}{})
	requireFoundInList(t, namespaces, func(ns godo.FunctionsNamespace) bool {
		return ns.Namespace == namespace.Namespace
	}, "namespace")
	t.Logf("found namespace in list (total: %d)", len(namespaces))
}

// TestServerlessFunctionLifecycle tests the full lifecycle of a serverless function:
// create namespace -> create function -> get function -> list functions -> invoke function -> delete function
func TestServerlessFunctionLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	namespaceLabel := fmt.Sprintf("mcp-e2e-fn-%d", time.Now().Unix())

	// create namespace for the test
	t.Log("creating serverless namespace for function tests...")
	namespace := callTool[godo.FunctionsNamespace](t, "serverless-namespace-create", map[string]interface{}{
		"Label":  namespaceLabel,
		"Region": testRegionNYC1,
	})
	require.NotEmpty(t, namespace.Namespace)
	t.Logf("created namespace: %s", namespace.Namespace)

	defer func() { deleteNamespace(t, tc, namespace.Namespace) }()

	functionName := "hello"
	functionCode := `function main(args) { return { body: "Hello, " + (args.name || "World") + "!" }; }`

	// create function
	t.Log("creating serverless function...")
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-create",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
				"Kind":         "nodejs:22",
				"Code":         functionCode,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "create function should succeed: %v", resp.Content)
	t.Log("created serverless function")

	defer func() {
		t.Log("deleting serverless function...")
		delResp, delErr := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "serverless-function-delete",
				Arguments: map[string]interface{}{
					"NamespaceID":  namespace.Namespace,
					"FunctionName": functionName,
				},
			},
		})
		if delErr != nil || (delResp != nil && delResp.IsError) {
			t.Logf("warning: failed to delete function: err=%v", delErr)
		}
	}()

	// get function
	t.Log("getting serverless function...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-get",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "get function should succeed: %v", resp.Content)

	var fnResult map[string]interface{}
	fnJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(fnJSON), &fnResult)
	require.NoError(t, err)
	require.Equal(t, functionName, fnResult["name"])
	t.Logf("fetched function: %s", fnResult["name"])

	// list functions
	t.Log("listing serverless functions...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-list",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "list functions should succeed: %v", resp.Content)

	var functions []map[string]interface{}
	fnListJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(fnListJSON), &functions)
	require.NoError(t, err)
	require.NotEmpty(t, functions, "should have at least one function")
	t.Logf("found %d functions", len(functions))

	// invoke function
	t.Log("invoking serverless function...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-invoke",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
				"Params":       map[string]interface{}{"name": "MCP"},
				"Blocking":     true,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "invoke function should succeed: %v", resp.Content)

	invokeText := resp.Content[0].(mcp.TextContent).Text
	require.NotEmpty(t, invokeText, "invoke result should not be empty")
	t.Logf("invoke result: %s", invokeText)
}

// TestServerlessTriggerLifecycle tests the full lifecycle of a serverless trigger:
// create namespace -> create function -> create trigger -> get trigger -> list triggers -> update trigger -> delete trigger
func TestServerlessTriggerLifecycle(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	namespaceLabel := fmt.Sprintf("mcp-e2e-trg-%d", time.Now().Unix())

	// create namespace
	t.Log("creating serverless namespace for trigger tests...")
	namespace := callTool[godo.FunctionsNamespace](t, "serverless-namespace-create", map[string]interface{}{
		"Label":  namespaceLabel,
		"Region": testRegionNYC1,
	})
	require.NotEmpty(t, namespace.Namespace)
	t.Logf("created namespace: %s", namespace.Namespace)

	defer func() { deleteNamespace(t, tc, namespace.Namespace) }()

	// create a function to attach the trigger to
	functionName := "trigger-target"
	functionCode := `function main(args) { return { body: "triggered" }; }`

	t.Log("creating target function for trigger...")
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-create",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
				"Kind":         "nodejs:22",
				"Code":         functionCode,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "create function should succeed: %v", resp.Content)
	t.Log("created target function")

	triggerName := fmt.Sprintf("mcp-e2e-trigger-%d", time.Now().Unix())

	// create trigger
	t.Log("creating serverless trigger...")
	trigger := callTool[godo.FunctionsTrigger](t, "serverless-trigger-create", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"Name":        triggerName,
		"Function":    functionName,
		"Type":        "SCHEDULED",
		"IsEnabled":   false,
		"Cron":        "0 * * * *",
	})
	require.Equal(t, triggerName, trigger.Name)
	t.Logf("created trigger: %s", trigger.Name)

	defer func() {
		t.Logf("deleting trigger %s...", triggerName)
		delResp, delErr := c.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "serverless-trigger-delete",
				Arguments: map[string]interface{}{
					"NamespaceID": namespace.Namespace,
					"TriggerName": triggerName,
				},
			},
		})
		if delErr != nil || (delResp != nil && delResp.IsError) {
			t.Logf("warning: failed to delete trigger: err=%v", delErr)
		}
	}()

	// get trigger
	t.Log("getting serverless trigger...")
	fetchedTrigger := callTool[godo.FunctionsTrigger](t, "serverless-trigger-get", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"TriggerName": triggerName,
	})
	require.Equal(t, triggerName, fetchedTrigger.Name)
	t.Logf("fetched trigger: %s (enabled: %v)", fetchedTrigger.Name, fetchedTrigger.IsEnabled)

	// list triggers
	t.Log("listing serverless triggers...")
	triggers := callTool[[]godo.FunctionsTrigger](t, "serverless-trigger-list", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
	})
	requireFoundInList(t, triggers, func(tr godo.FunctionsTrigger) bool {
		return tr.Name == triggerName
	}, "trigger")
	t.Logf("found trigger in list (total: %d)", len(triggers))

	// update trigger
	t.Log("updating serverless trigger...")
	updatedTrigger := callTool[godo.FunctionsTrigger](t, "serverless-trigger-update", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"TriggerName": triggerName,
		"IsEnabled":   true,
		"Cron":        "*/30 * * * *",
	})
	require.Equal(t, triggerName, updatedTrigger.Name)
	t.Logf("updated trigger: %s (enabled: %v)", updatedTrigger.Name, updatedTrigger.IsEnabled)
}

// TestServerlessActivations tests listing activations for a namespace.
func TestServerlessActivations(t *testing.T) {
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	namespaceLabel := fmt.Sprintf("mcp-e2e-act-%d", time.Now().Unix())

	// create namespace
	t.Log("creating serverless namespace for activation tests...")
	namespace := callTool[godo.FunctionsNamespace](t, "serverless-namespace-create", map[string]interface{}{
		"Label":  namespaceLabel,
		"Region": testRegionNYC1,
	})
	require.NotEmpty(t, namespace.Namespace)
	t.Logf("created namespace: %s", namespace.Namespace)

	defer func() { deleteNamespace(t, tc, namespace.Namespace) }()

	// create and invoke a function to generate an activation
	functionName := "activation-test"
	functionCode := `function main(args) { return { body: "ok" }; }`

	t.Log("creating function for activation test...")
	resp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-create",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
				"Kind":         "nodejs:22",
				"Code":         functionCode,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "create function should succeed: %v", resp.Content)

	// invoke to generate an activation
	t.Log("invoking function to generate activation...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-function-invoke",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"FunctionName": functionName,
				"Blocking":     true,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "invoke should succeed: %v", resp.Content)
	t.Log("function invoked successfully")

	// list activations
	t.Log("listing activations...")
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-activation-list",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"Limit":       float64(10),
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "list activations should succeed: %v", resp.Content)

	var activations []map[string]interface{}
	actJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(actJSON), &activations)
	require.NoError(t, err)
	require.NotEmpty(t, activations, "should have at least one activation after invoke")
	t.Logf("found %d activations", len(activations))

	// get a specific activation
	activationID, ok := activations[0]["activationId"].(string)
	require.True(t, ok, "activation should have an activationId")

	t.Logf("getting activation %s...", activationID)
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-activation-get",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"ActivationID": activationID,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "get activation should succeed: %v", resp.Content)
	t.Logf("fetched activation details for %s", activationID)

	// get activation logs
	t.Logf("getting activation logs for %s...", activationID)
	resp, err = c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "serverless-activation-logs",
			Arguments: map[string]interface{}{
				"NamespaceID":  namespace.Namespace,
				"ActivationID": activationID,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "get activation logs should succeed: %v", resp.Content)
	t.Log("fetched activation logs")
}
