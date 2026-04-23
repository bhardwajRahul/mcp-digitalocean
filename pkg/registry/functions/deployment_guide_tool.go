package functions

import (
	"context"
	_ "embed"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed DEPLOY_SPEC.md
var deploymentGuideContent string

// DeploymentGuideTool exposes the authoritative agent-facing spec for
// deploying a DigitalOcean Functions project from a local directory via
// `doctl serverless deploy`.
//
// The tool returns the full spec as markdown. Agents are expected to call
// this tool once at the start of a deploy flow and follow it step by step.
type DeploymentGuideTool struct{}

func NewDeploymentGuideTool() *DeploymentGuideTool {
	return &DeploymentGuideTool{}
}

func (t *DeploymentGuideTool) getDeploymentGuide(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(deploymentGuideContent), nil
}

func (t *DeploymentGuideTool) Tools() []server.ServerTool {
	return []server.ServerTool{
		{
			Handler: t.getDeploymentGuide,
			Tool: mcp.NewTool("functions-deployment-guide",
				mcp.WithDescription(
					"Return the authoritative step-by-step guide for deploying a DigitalOcean "+
						"Functions project from a local directory using `doctl serverless deploy`.\n\n"+
						"Call this tool FIRST when the user asks to deploy a function, update a deployed "+
						"function's code, or set up a new Functions project from scratch. The guide covers:\n"+
						"- Preflight checks (`doctl` install, `doctl auth init`, serverless plugin install)\n"+
						"- Namespace selection/creation and `doctl serverless connect` (including the "+
						"access-key fallback)\n"+
						"- Project scaffolding (single-file, with-dependencies, multi-function layouts) "+
						"and when to prefer `doctl serverless init`\n"+
						"- The deploy command itself and the local-vs-remote-build decision\n"+
						"- Post-deploy verification via the other `functions-*` MCP tools\n"+
						"- Supported runtimes, stop conditions, and error-handling rules\n\n"+
						"Do NOT call this tool for routine per-action CRUD operations "+
						"(create/update/delete/invoke a single action from inline code) — use the "+
						"`functions-create-or-update-action` and related tools directly for those.\n\n"+
						"The returned content is markdown. Follow its instructions exactly; do not "+
						"paraphrase the steps to the user.",
				),
				mcp.WithReadOnlyHintAnnotation(true),
			),
		},
	}
}
