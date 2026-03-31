package serverless

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	middleware "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	owActionsBase     = "/api/v1/namespaces/_/actions"
	owActivationsBase = "/api/v1/namespaces/_/activations"
	defaultFnLimit    = 100
	defaultFnSkip     = 0
	defaultOverwrite  = true
)

// namespaceInfo holds the resolved OpenWhisk API host and auth key for a namespace.
type namespaceInfo struct {
	apiHost string
	key     string
}

// FunctionTool provides serverless function management tools via the OpenWhisk API.
type FunctionTool struct {
	client       func(ctx context.Context) (*godo.Client, error)
	httpClient   *http.Client
	cache        *accessKeyCache
	accessKeySvc AccessKeyService
}

// FunctionToolOption configures a FunctionTool.
type FunctionToolOption func(*FunctionTool)

// WithAccessKeyService sets the access key service for OpenWhisk authentication.
// When set, resolveNamespace will create and cache access keys instead of using
// the deprecated namespace UUID:Key fields.
func WithAccessKeyService(svc AccessKeyService) FunctionToolOption {
	return func(f *FunctionTool) {
		f.accessKeySvc = svc
	}
}

// NewFunctionTool creates a new FunctionTool.
// The ctx parameter controls the lifetime of the background cache cleanup goroutine.
func NewFunctionTool(ctx context.Context, client func(ctx context.Context) (*godo.Client, error), opts ...FunctionToolOption) *FunctionTool {
	f := &FunctionTool{
		client:     client,
		httpClient: http.DefaultClient,
		cache:      newAccessKeyCache(ctx),
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// resolveNamespace returns the OpenWhisk API host and auth key for a namespace.
// It first attempts to use access keys (cached or newly created) if an
// AccessKeyService is configured, then falls back to the legacy UUID:Key path.
func (f *FunctionTool) resolveNamespace(ctx context.Context, namespaceID string) (*namespaceInfo, error) {
	if f.accessKeySvc != nil {
		info, err := f.resolveNamespaceViaAccessKey(ctx, namespaceID)
		if err == nil {
			return info, nil
		}
		// Fall through to legacy path on access key failure.
	}
	return f.resolveNamespaceLegacy(ctx, namespaceID)
}

// resolveNamespaceViaAccessKey resolves namespace credentials using the access key
// cache and service. It hashes the caller's bearer token (from context) to scope
// cache entries per user in multi-tenant SSE mode.
func (f *FunctionTool) resolveNamespaceViaAccessKey(ctx context.Context, namespaceID string) (*namespaceInfo, error) {
	tokenHash := extractTokenHash(ctx)

	if cached := f.cache.get(tokenHash, namespaceID); cached != nil {
		return &namespaceInfo{
			apiHost: cached.apiHost,
			key:     cached.id + ":" + cached.secret,
		}, nil
	}

	ak, err := f.accessKeySvc.CreateAccessKey(ctx, namespaceID, &AccessKeyCreateRequest{
		Name:      accessKeyClientName,
		ExpiresIn: accessKeyTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create access key: %w", err)
	}

	apiHost := strings.TrimRight(ak.APIHost, "/")
	f.cache.put(tokenHash, namespaceID, &cachedAccessKey{
		id:        ak.ID,
		secret:    ak.Secret,
		apiHost:   apiHost,
		expiresAt: ak.ExpiresAt,
	})

	return &namespaceInfo{
		apiHost: apiHost,
		key:     ak.ID + ":" + ak.Secret,
	}, nil
}

// resolveNamespaceLegacy fetches the namespace via godo and uses the deprecated
// UUID:Key fields for OpenWhisk basic auth. This path is used when no
// AccessKeyService is configured or when access key creation fails.
func (f *FunctionTool) resolveNamespaceLegacy(ctx context.Context, namespaceID string) (*namespaceInfo, error) {
	client, err := f.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	ns, _, err := client.Functions.GetNamespace(ctx, namespaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	if ns.ApiHost == "" || ns.Key == "" || ns.UUID == "" {
		return nil, fmt.Errorf("namespace %s is missing api_host, uuid, or key", namespaceID)
	}

	return &namespaceInfo{
		apiHost: strings.TrimRight(ns.ApiHost, "/"),
		key:     ns.UUID + ":" + ns.Key,
	}, nil
}

// extractTokenHash returns a hashed version of the bearer token from context.
// Returns an empty string when no token is present (e.g., stdio transport).
func extractTokenHash(ctx context.Context) string {
	auth, ok := ctx.Value(middleware.AuthKey{}).(string)
	if !ok || auth == "" {
		return ""
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" {
		return ""
	}
	return hashToken(token)
}

// owRequest makes an authenticated HTTP request to the OpenWhisk API.
func (f *FunctionTool) owRequest(ctx context.Context, method, url, key string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		req.SetBasicAuth(parts[0], parts[1])
	} else {
		req.SetBasicAuth(key, "")
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

// list lists all functions in a namespace.
func (f *FunctionTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	limit := defaultFnLimit
	skip := defaultFnSkip
	if v, ok := req.GetArguments()["Limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if v, ok := req.GetArguments()["Skip"].(float64); ok && int(v) >= 0 {
		skip = int(v)
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s?limit=%d&skip=%d", ns.apiHost, owActionsBase, limit, skip)
	body, status, err := f.owRequest(ctx, http.MethodGet, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// get fetches a single function by name.
func (f *FunctionTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	functionName, ok := req.GetArguments()["FunctionName"].(string)
	if !ok || functionName == "" {
		return mcp.NewToolResultError("FunctionName is required"), nil
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s", ns.apiHost, owActionsBase, functionName)
	body, status, err := f.owRequest(ctx, http.MethodGet, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// create creates or updates a function with inline code.
func (f *FunctionTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	functionName, ok := req.GetArguments()["FunctionName"].(string)
	if !ok || functionName == "" {
		return mcp.NewToolResultError("FunctionName is required"), nil
	}

	kind, ok := req.GetArguments()["Kind"].(string)
	if !ok || kind == "" {
		return mcp.NewToolResultError("Kind is required (e.g., 'nodejs:18', 'python:3.11', 'go:1.21', 'php:8.2')"), nil
	}

	code, ok := req.GetArguments()["Code"].(string)
	if !ok || code == "" {
		return mcp.NewToolResultError("Code is required"), nil
	}

	overwrite := defaultOverwrite
	if v, ok := req.GetArguments()["Overwrite"].(bool); ok {
		overwrite = v
	}

	actionBody := map[string]interface{}{
		"namespace": "_",
		"name":      functionName,
		"exec": map[string]interface{}{
			"kind": kind,
			"code": code,
		},
	}

	if webExport, ok := req.GetArguments()["WebExport"].(bool); ok && webExport {
		actionBody["annotations"] = []map[string]interface{}{
			{"key": "web-export", "value": true},
			{"key": "final", "value": true},
		}
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s?overwrite=%t", ns.apiHost, owActionsBase, functionName, overwrite)
	body, status, err := f.owRequest(ctx, http.MethodPut, url, ns.key, actionBody)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// delete deletes a function.
func (f *FunctionTool) deleteFn(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	functionName, ok := req.GetArguments()["FunctionName"].(string)
	if !ok || functionName == "" {
		return mcp.NewToolResultError("FunctionName is required"), nil
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s", ns.apiHost, owActionsBase, functionName)
	body, status, err := f.owRequest(ctx, http.MethodDelete, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	return mcp.NewToolResultText("function deleted successfully"), nil
}

// invoke invokes a function and returns the result.
func (f *FunctionTool) invoke(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	functionName, ok := req.GetArguments()["FunctionName"].(string)
	if !ok || functionName == "" {
		return mcp.NewToolResultError("FunctionName is required"), nil
	}

	blocking := true
	if v, ok := req.GetArguments()["Blocking"].(bool); ok {
		blocking = v
	}

	params := map[string]interface{}{}
	if p, ok := req.GetArguments()["Params"].(map[string]interface{}); ok {
		params = p
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s?blocking=%t&result=true", ns.apiHost, owActionsBase, functionName, blocking)
	body, status, err := f.owRequest(ctx, http.MethodPost, url, ns.key, params)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK && status != http.StatusAccepted {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// listActivations lists activation records for a namespace.
func (f *FunctionTool) listActivations(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	limit := defaultFnLimit
	skip := defaultFnSkip
	if v, ok := req.GetArguments()["Limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if v, ok := req.GetArguments()["Skip"].(float64); ok && int(v) >= 0 {
		skip = int(v)
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s?limit=%d&skip=%d", ns.apiHost, owActivationsBase, limit, skip)

	if fnName, ok := req.GetArguments()["FunctionName"].(string); ok && fnName != "" {
		url += "&name=" + fnName
	}
	if since, ok := req.GetArguments()["Since"].(float64); ok && int64(since) > 0 {
		url += fmt.Sprintf("&since=%d", int64(since))
	}
	if upto, ok := req.GetArguments()["Upto"].(float64); ok && int64(upto) > 0 {
		url += fmt.Sprintf("&upto=%d", int64(upto))
	}

	body, status, err := f.owRequest(ctx, http.MethodGet, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// getActivation fetches a single activation record by ID.
func (f *FunctionTool) getActivation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	activationID, ok := req.GetArguments()["ActivationID"].(string)
	if !ok || activationID == "" {
		return mcp.NewToolResultError("ActivationID is required"), nil
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s", ns.apiHost, owActivationsBase, activationID)
	body, status, err := f.owRequest(ctx, http.MethodGet, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// getActivationLogs fetches the logs for a specific activation.
func (f *FunctionTool) getActivationLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	activationID, ok := req.GetArguments()["ActivationID"].(string)
	if !ok || activationID == "" {
		return mcp.NewToolResultError("ActivationID is required"), nil
	}

	ns, err := f.resolveNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("namespace error", err), nil
	}

	url := fmt.Sprintf("%s%s/%s/logs", ns.apiHost, owActivationsBase, activationID)
	body, status, err := f.owRequest(ctx, http.MethodGet, url, ns.key, nil)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}
	if status != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("OpenWhisk API error (status %d): %s", status, string(body))), nil
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		return mcp.NewToolResultText(string(body)), nil
	}

	return mcp.NewToolResultText(prettyJSON.String()), nil
}

// Tools returns a list of tool functions for function management.
func (f *FunctionTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: f.list,
			Tool: mcp.NewTool("serverless-function-list",
				mcp.WithDescription("List all functions in a serverless namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithNumber("Limit", mcp.Description("Maximum number of functions to return (default: 100)")),
				mcp.WithNumber("Skip", mcp.Description("Number of functions to skip for pagination (default: 0)")),
			),
		},
		{
			Handler: f.get,
			Tool: mcp.NewTool("serverless-function-get",
				mcp.WithDescription("Get details of a specific function in a serverless namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("FunctionName", mcp.Required(), mcp.Description("Name of the function (e.g., 'mypackage/myfunction' or 'myfunction')")),
			),
		},
		{
			Handler: f.create,
			Tool: mcp.NewTool("serverless-function-create",
				mcp.WithDescription("Create or update a serverless function with inline code"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("FunctionName", mcp.Required(), mcp.Description("Name of the function (e.g., 'mypackage/myfunction' or 'myfunction')")),
				mcp.WithString("Kind", mcp.Required(), mcp.Description("Runtime kind (e.g., 'nodejs:22', 'python:3.12', 'go:1.21', 'php:8.2')")),
				mcp.WithString("Code", mcp.Required(), mcp.Description("Source code of the function")),
				mcp.WithBoolean("WebExport", mcp.Description("Make the function publicly accessible as a web action (default: false)")),
				mcp.WithBoolean("Overwrite", mcp.Description("Overwrite the function if it already exists (default: true)")),
			),
		},
		{
			Handler: f.deleteFn,
			Tool: mcp.NewTool("serverless-function-delete",
				mcp.WithDescription("Delete a function from a serverless namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("FunctionName", mcp.Required(), mcp.Description("Name of the function to delete")),
			),
		},
		{
			Handler: f.invoke,
			Tool: mcp.NewTool("serverless-function-invoke",
				mcp.WithDescription("Invoke a serverless function and return the result"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("FunctionName", mcp.Required(), mcp.Description("Name of the function to invoke")),
				mcp.WithObject("Params", mcp.Description("JSON parameters to pass to the function")),
				mcp.WithBoolean("Blocking", mcp.Description("Wait for the function to complete before returning (default: true)")),
			),
		},
		{
			Handler: f.listActivations,
			Tool: mcp.NewTool("serverless-activation-list",
				mcp.WithDescription("List activation records for a serverless namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("FunctionName", mcp.Description("Filter activations by function name")),
				mcp.WithNumber("Limit", mcp.Description("Maximum number of activations to return (default: 100)")),
				mcp.WithNumber("Skip", mcp.Description("Number of activations to skip for pagination (default: 0)")),
				mcp.WithNumber("Since", mcp.Description("Only return activations after this timestamp (epoch milliseconds)")),
				mcp.WithNumber("Upto", mcp.Description("Only return activations before this timestamp (epoch milliseconds)")),
			),
		},
		{
			Handler: f.getActivation,
			Tool: mcp.NewTool("serverless-activation-get",
				mcp.WithDescription("Get details of a specific activation record"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("ActivationID", mcp.Required(), mcp.Description("ID of the activation")),
			),
		},
		{
			Handler: f.getActivationLogs,
			Tool: mcp.NewTool("serverless-activation-logs",
				mcp.WithDescription("Get logs for a specific function activation"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("ActivationID", mcp.Required(), mcp.Description("ID of the activation")),
			),
		},
	}
}
