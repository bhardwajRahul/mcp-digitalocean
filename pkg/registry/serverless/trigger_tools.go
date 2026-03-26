package serverless

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TriggerTool provides serverless trigger management tools
type TriggerTool struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewTriggerTool creates a new TriggerTool
func NewTriggerTool(client func(ctx context.Context) (*godo.Client, error)) *TriggerTool {
	return &TriggerTool{
		client: client,
	}
}

// list lists all triggers for a namespace
func (t *TriggerTool) list(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	triggers, _, err := client.Functions.ListTriggers(ctx, namespaceID)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonTriggers, err := json.MarshalIndent(triggers, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonTriggers)), nil
}

// get fetches a single trigger by name
func (t *TriggerTool) get(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	triggerName, ok := req.GetArguments()["TriggerName"].(string)
	if !ok || triggerName == "" {
		return mcp.NewToolResultError("TriggerName is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.GetTrigger(ctx, namespaceID, triggerName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonTrigger, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonTrigger)), nil
}

// create creates a new trigger
func (t *TriggerTool) create(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	name, ok := req.GetArguments()["Name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("Name is required"), nil
	}

	function, ok := req.GetArguments()["Function"].(string)
	if !ok || function == "" {
		return mcp.NewToolResultError("Function is required"), nil
	}

	triggerType, ok := req.GetArguments()["Type"].(string)
	if !ok || triggerType == "" {
		return mcp.NewToolResultError("Type is required"), nil
	}

	isEnabled, _ := req.GetArguments()["IsEnabled"].(bool)

	createReq := &godo.FunctionsTriggerCreateRequest{
		Name:      name,
		Function:  function,
		Type:      triggerType,
		IsEnabled: isEnabled,
	}

	if cron, ok := req.GetArguments()["Cron"].(string); ok && cron != "" {
		createReq.ScheduledDetails = &godo.TriggerScheduledDetails{
			Cron: cron,
		}
		if body, ok := req.GetArguments()["Body"].(map[string]interface{}); ok {
			createReq.ScheduledDetails.Body = body
		}
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.CreateTrigger(ctx, namespaceID, createReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonTrigger, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonTrigger)), nil
}

// update updates an existing trigger
func (t *TriggerTool) update(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	triggerName, ok := req.GetArguments()["TriggerName"].(string)
	if !ok || triggerName == "" {
		return mcp.NewToolResultError("TriggerName is required"), nil
	}

	updateReq := &godo.FunctionsTriggerUpdateRequest{}

	if isEnabled, ok := req.GetArguments()["IsEnabled"].(bool); ok {
		updateReq.IsEnabled = &isEnabled
	}

	if cron, ok := req.GetArguments()["Cron"].(string); ok && cron != "" {
		updateReq.ScheduledDetails = &godo.TriggerScheduledDetails{
			Cron: cron,
		}
		if body, ok := req.GetArguments()["Body"].(map[string]interface{}); ok {
			updateReq.ScheduledDetails.Body = body
		}
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	trigger, _, err := client.Functions.UpdateTrigger(ctx, namespaceID, triggerName, updateReq)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	jsonTrigger, err := json.MarshalIndent(trigger, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}

	return mcp.NewToolResultText(string(jsonTrigger)), nil
}

// delete deletes a trigger
func (t *TriggerTool) delete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespaceID, ok := req.GetArguments()["NamespaceID"].(string)
	if !ok || namespaceID == "" {
		return mcp.NewToolResultError("NamespaceID is required"), nil
	}

	triggerName, ok := req.GetArguments()["TriggerName"].(string)
	if !ok || triggerName == "" {
		return mcp.NewToolResultError("TriggerName is required"), nil
	}

	client, err := t.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DigitalOcean client: %w", err)
	}

	_, err = client.Functions.DeleteTrigger(ctx, namespaceID, triggerName)
	if err != nil {
		return mcp.NewToolResultErrorFromErr("api error", err), nil
	}

	return mcp.NewToolResultText("trigger deleted successfully"), nil
}

// Tools returns a list of tool functions for trigger management
func (t *TriggerTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.list,
			Tool: mcp.NewTool("serverless-trigger-list",
				mcp.WithDescription("List all triggers for a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
			),
		},
		{
			Handler: t.get,
			Tool: mcp.NewTool("serverless-trigger-get",
				mcp.WithDescription("Get a trigger by name for a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("Name of the trigger")),
			),
		},
		{
			Handler: t.create,
			Tool: mcp.NewTool("serverless-trigger-create",
				mcp.WithDescription("Create a new trigger for a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("Name", mcp.Required(), mcp.Description("Name of the trigger")),
				mcp.WithString("Function", mcp.Required(), mcp.Description("Function to invoke (e.g., 'mypackage/myfunction')")),
				mcp.WithString("Type", mcp.Required(), mcp.Description("Type of trigger (e.g., 'SCHEDULED')")),
				mcp.WithBoolean("IsEnabled", mcp.Description("Whether the trigger is enabled (default: false)")),
				mcp.WithString("Cron", mcp.Description("Cron expression for scheduled triggers (e.g., '*/5 * * * *')")),
				mcp.WithObject("Body", mcp.Description("JSON body to pass to the function when triggered")),
			),
		},
		{
			Handler: t.update,
			Tool: mcp.NewTool("serverless-trigger-update",
				mcp.WithDescription("Update a trigger for a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("Name of the trigger to update")),
				mcp.WithBoolean("IsEnabled", mcp.Description("Whether the trigger is enabled")),
				mcp.WithString("Cron", mcp.Description("Updated cron expression for scheduled triggers")),
				mcp.WithObject("Body", mcp.Description("Updated JSON body to pass to the function when triggered")),
			),
		},
		{
			Handler: t.delete,
			Tool: mcp.NewTool("serverless-trigger-delete",
				mcp.WithDescription("Delete a trigger from a serverless function namespace"),
				mcp.WithString("NamespaceID", mcp.Required(), mcp.Description("The 'namespace' field from the namespace object (e.g., 'fn-abc123-...')")),
				mcp.WithString("TriggerName", mcp.Required(), mcp.Description("Name of the trigger to delete")),
			),
		},
	}
}
