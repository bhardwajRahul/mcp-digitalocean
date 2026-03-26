package serverless

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NamespaceTool provides serverless namespace management tools
type NamespaceTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewNamespaceTool creates a new NamespaceTool
func NewNamespaceTool(client func(ctx context.Context) (*godo.Client, error)) *NamespaceTool {
	return &NamespaceTool{
		client: client,
	}
}

// list lists all serverless namespaces
func (n *NamespaceTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := n.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	namespaces, _, err := client.Functions.ListNamespaces(ctx)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNamespaces, err := json.MarshalIndent(namespaces, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNamespaces)), nil
}

// get fetches a serverless namespace by ID
func (n *NamespaceTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	namespace, _, err := client.Functions.GetNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNamespace, err := json.MarshalIndent(namespace, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNamespace)), nil
}

// create creates a new serverless namespace
func (n *NamespaceTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	label, ok := req.GetArguments()["Label"].(string)
	if !ok || label == "" {
		return mcp.NewToolResultError("Label is required"), nil
	}

	region, ok := req.GetArguments()["Region"].(string)
	if !ok || region == "" {
		return mcp.NewToolResultError("Region is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	namespace, _, err := client.Functions.CreateNamespace(ctx, &godo.FunctionsNamespaceCreateRequest{
		Label:  label,
		Region: region,
	})
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonNamespace, err := json.MarshalIndent(namespace, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonNamespace)), nil
}

// delete deletes a serverless namespace
func (n *NamespaceTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	client, err := n.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Functions.DeleteNamespace(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("namespace deleted successfully"), nil
}

// Tools returns a list of tool functions for namespace management
func (n *NamespaceTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: n.list,
			Tool: mcp.NewTool("serverless-namespace-list",
				mcp.WithDescription("List all serverless function namespaces"),
			),
		},
		{
			Handler: n.get,
			Tool: mcp.NewTool("serverless-namespace-get",
				mcp.WithDescription("Get a serverless function namespace by ID"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
			),
		},
		{
			Handler: n.create,
			Tool: mcp.NewTool("serverless-namespace-create",
				mcp.WithDescription("Create a new serverless function namespace"),
				mcp.WithString("Label", mcp.Required(), mcp.Description("Label for the namespace")),
				mcp.WithString("Region", mcp.Required(), mcp.Description("Region slug for the namespace (e.g., 'nyc1', 'sfo3')")),
			),
		},
		{
			Handler: n.delete,
			Tool: mcp.NewTool("serverless-namespace-delete",
				mcp.WithDescription("Delete a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
			),
		},
	}
}
