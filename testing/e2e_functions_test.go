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
	t.Logf("deleting functions namespace %s...", namespaceID)
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "functions-delete-namespace",
			Arguments: map[string]interface{}{"NamespaceID": namespaceID},
		},
	})
	if err != nil {
		t.Logf("failed to delete namespace: %v", err)
		return
	}
	if resp.IsError {
		t.Logf("functions-delete-namespace returned error: %v", resp.Content)
		return
	}
	t.Logf("deleted namespace %s", namespaceID)
}

// createTestNamespace creates a namespace and registers cleanup.
func createTestNamespace(t *testing.T, label string) (godo.FunctionsNamespace, testContext) {
	t.Helper()
	ctx, c := setupTest(t)
	tc := testContext{ctx: ctx, client: c}

	ns := callTool[godo.FunctionsNamespace](t, "functions-create-namespace", map[string]interface{}{
		"Label":  label,
		"Region": testRegionNYC1,
	})
	require.NotEmpty(t, ns.Namespace, "namespace ID should not be empty")
	t.Logf("created namespace: %s (ID: %s)", ns.Label, ns.Namespace)

	t.Cleanup(func() { deleteNamespace(t, tc, ns.Namespace) })
	return ns, tc
}

// createTestAction creates an action in the given namespace and registers cleanup.
func createTestAction(t *testing.T, tc testContext, namespaceID, actionName, code string) {
	t.Helper()
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-create-or-update-action",
			Arguments: map[string]interface{}{
				"NamespaceID": namespaceID,
				"ActionName":  actionName,
				"Kind":        "nodejs:22",
				"Code":        code,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.False(t, resp.IsError, "create action should succeed: %v", resp.Content)
	t.Logf("created action: %s", actionName)

	t.Cleanup(func() {
		delResp, delErr := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "functions-delete-action",
				Arguments: map[string]interface{}{
					"NamespaceID": namespaceID,
					"ActionName":  actionName,
				},
			},
		})
		if delErr != nil || (delResp != nil && delResp.IsError) {
			t.Logf("warning: failed to delete action %s: err=%v", actionName, delErr)
		}
	})
}

// TestFunctionsNamespaceLifecycle tests create -> get -> list -> delete.
func TestFunctionsNamespaceLifecycle(t *testing.T) {
	namespaceLabel := fmt.Sprintf("mcp-e2e-ns-%d", time.Now().Unix())
	namespace, _ := createTestNamespace(t, namespaceLabel)

	// get namespace
	t.Log("getting namespace...")
	fetched := callTool[godo.FunctionsNamespace](t, "functions-get-namespace", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
	})
	require.Equal(t, namespace.Namespace, fetched.Namespace)
	require.Equal(t, namespaceLabel, fetched.Label)
	t.Logf("fetched namespace: %s (label: %s)", fetched.Namespace, fetched.Label)

	// list namespaces
	t.Log("listing namespaces...")
	namespaces := callTool[[]godo.FunctionsNamespace](t, "functions-list-namespaces", map[string]interface{}{})
	requireFoundInList(t, namespaces, func(ns godo.FunctionsNamespace) bool {
		return ns.Namespace == namespace.Namespace
	}, "namespace")
	t.Logf("found namespace in list (total: %d)", len(namespaces))
}

// TestFunctionsActionLifecycle tests create -> get -> list -> invoke -> delete.
func TestFunctionsActionLifecycle(t *testing.T) {
	namespaceLabel := fmt.Sprintf("mcp-e2e-fn-%d", time.Now().Unix())
	namespace, tc := createTestNamespace(t, namespaceLabel)

	actionName := "hello"
	actionCode := `function main(args) { return { body: "Hello, " + (args.name || "World") + "!" }; }`
	createTestAction(t, tc, namespace.Namespace, actionName, actionCode)

	// get action
	t.Log("getting action...")
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-get-action",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"ActionName":  actionName,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "get action should succeed: %v", resp.Content)

	var fnResult map[string]interface{}
	fnJSON := resp.Content[0].(mcp.TextContent).Text
	err = json.Unmarshal([]byte(fnJSON), &fnResult)
	require.NoError(t, err)
	require.Equal(t, actionName, fnResult["name"])
	t.Logf("fetched action: %s", fnResult["name"])

	// list actions
	t.Log("listing actions...")
	resp, err = tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-list-actions",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "list actions should succeed: %v", resp.Content)

	var actions []map[string]interface{}
	err = json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &actions)
	require.NoError(t, err)
	require.NotEmpty(t, actions, "should have at least one action")
	t.Logf("found %d actions", len(actions))

	// invoke action
	t.Log("invoking action...")
	resp, err = tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-invoke-action",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"ActionName":  actionName,
				"Payload":     map[string]interface{}{"name": "MCP"},
				"Blocking":    true,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "invoke action should succeed: %v", resp.Content)

	invokeText := resp.Content[0].(mcp.TextContent).Text
	require.NotEmpty(t, invokeText, "invoke result should not be empty")
	t.Logf("invoke result: %s", invokeText)
}

// TestFunctionsPackageLifecycle tests create -> get -> list -> delete for packages.
func TestFunctionsPackageLifecycle(t *testing.T) {
	namespaceLabel := fmt.Sprintf("mcp-e2e-pkg-%d", time.Now().Unix())
	namespace, tc := createTestNamespace(t, namespaceLabel)

	pkgName := "mcp-test-pkg"

	// create package
	t.Log("creating package...")
	resp, err := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-create-or-update-package",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"PackageName": pkgName,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "create package should succeed: %v", resp.Content)
	t.Logf("created package: %s", pkgName)

	defer func() {
		t.Logf("deleting package %s...", pkgName)
		delResp, delErr := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "functions-delete-package",
				Arguments: map[string]interface{}{
					"NamespaceID": namespace.Namespace,
					"PackageName": pkgName,
					"Force":       true,
				},
			},
		})
		if delErr != nil || (delResp != nil && delResp.IsError) {
			t.Logf("warning: failed to delete package: err=%v", delErr)
		}
	}()

	// get package
	t.Log("getting package...")
	resp, err = tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-get-package",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"PackageName": pkgName,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "get package should succeed: %v", resp.Content)

	var pkgResult map[string]interface{}
	err = json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &pkgResult)
	require.NoError(t, err)
	require.Equal(t, pkgName, pkgResult["name"])
	t.Logf("fetched package: %s", pkgResult["name"])

	// list packages
	t.Log("listing packages...")
	resp, err = tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-list-packages",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, resp.IsError, "list packages should succeed: %v", resp.Content)

	var packages []map[string]interface{}
	err = json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &packages)
	require.NoError(t, err)
	require.NotEmpty(t, packages, "should have at least one package")
	t.Logf("found %d packages", len(packages))
}

// TestFunctionsTriggerLifecycle tests create -> get -> list -> update -> delete.
func TestFunctionsTriggerLifecycle(t *testing.T) {
	namespaceLabel := fmt.Sprintf("mcp-e2e-trg-%d", time.Now().Unix())
	namespace, tc := createTestNamespace(t, namespaceLabel)

	actionName := "trigger-target"
	actionCode := `function main(args) { return { body: "triggered" }; }`
	createTestAction(t, tc, namespace.Namespace, actionName, actionCode)

	triggerName := fmt.Sprintf("mcp-e2e-trigger-%d", time.Now().Unix())

	// create trigger
	t.Log("creating trigger...")
	trigger := callTool[godo.FunctionsTrigger](t, "functions-create-trigger", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"Name":        triggerName,
		"Function":    actionName,
		"IsEnabled":   false,
		"Cron":        "0 * * * *",
	})
	require.Equal(t, triggerName, trigger.Name)
	t.Logf("created trigger: %s", trigger.Name)

	defer func() {
		t.Logf("deleting trigger %s...", triggerName)
		delResp, delErr := tc.client.CallTool(tc.ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "functions-delete-trigger",
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
	t.Log("getting trigger...")
	fetchedTrigger := callTool[godo.FunctionsTrigger](t, "functions-get-trigger", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"TriggerName": triggerName,
	})
	require.Equal(t, triggerName, fetchedTrigger.Name)
	t.Logf("fetched trigger: %s (enabled: %v)", fetchedTrigger.Name, fetchedTrigger.IsEnabled)

	// list triggers
	t.Log("listing triggers...")
	triggers := callTool[[]godo.FunctionsTrigger](t, "functions-list-triggers", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
	})
	requireFoundInList(t, triggers, func(tr godo.FunctionsTrigger) bool {
		return tr.Name == triggerName
	}, "trigger")
	t.Logf("found trigger in list (total: %d)", len(triggers))

	// update trigger
	t.Log("updating trigger...")
	updatedTrigger := callTool[godo.FunctionsTrigger](t, "functions-update-trigger", map[string]interface{}{
		"NamespaceID": namespace.Namespace,
		"TriggerName": triggerName,
		"IsEnabled":   true,
		"Cron":        "*/30 * * * *",
	})
	require.Equal(t, triggerName, updatedTrigger.Name)
	t.Logf("updated trigger: %s (enabled: %v)", updatedTrigger.Name, updatedTrigger.IsEnabled)
}

// TestFunctionsAccessKeys tests list -> create -> delete for access keys.
func TestFunctionsAccessKeys(t *testing.T) {
	namespaceLabel := fmt.Sprintf("mcp-e2e-ak-%d", time.Now().Unix())
	namespace, _ := createTestNamespace(t, namespaceLabel)

	// list access keys (should be empty or only mcp-do- internal keys)
	t.Log("listing access keys...")
	ctx, c := setupTest(t)

	listResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-list-access-keys",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, listResp.IsError, "list access keys should succeed: %v", listResp.Content)
	t.Log("listed access keys")

	// create access key
	keyName := fmt.Sprintf("mcp-e2e-key-%d", time.Now().Unix())
	t.Logf("creating access key %s...", keyName)

	createResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-create-access-key",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"Name":        keyName,
				"ExpiresIn":   "1h",
			},
		},
	})
	require.NoError(t, err)
	require.False(t, createResp.IsError, "create access key should succeed: %v", createResp.Content)

	var key map[string]interface{}
	err = json.Unmarshal([]byte(createResp.Content[0].(mcp.TextContent).Text), &key)
	require.NoError(t, err)
	keyID, ok := key["id"].(string)
	require.True(t, ok, "access key should have an id")
	require.NotEmpty(t, key["secret"], "access key should have a secret at creation time")
	t.Logf("created access key: %s", keyID)

	// delete access key
	t.Logf("deleting access key %s...", keyID)
	delResp, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "functions-delete-access-key",
			Arguments: map[string]interface{}{
				"NamespaceID": namespace.Namespace,
				"KeyID":       keyID,
			},
		},
	})
	require.NoError(t, err)
	require.False(t, delResp.IsError, "delete access key should succeed: %v", delResp.Content)
	t.Logf("deleted access key %s", keyID)
}
