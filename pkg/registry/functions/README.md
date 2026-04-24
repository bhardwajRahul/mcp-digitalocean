## DigitalOcean Functions Tools

This directory provides tools for managing DigitalOcean Functions (serverless) namespaces, actions, packages, triggers, activations, and access keys via the MCP Server. All operations are exposed as tools with argument-based input. Pagination and filtering are supported where applicable.

Two deploy paths — pick the right one:

- **Single file, no dependencies** → call `functions-create-or-update-action` directly with the source inline. No `doctl`, no project directory, no files left on disk.
- **Multi-file, any dependencies (`package.json` with deps / `requirements.txt` / `go.mod` / `composer.json`), a `build.sh`, or an existing `project.yml`** → call `functions-deployment-guide` first. It returns the full playbook for orchestrating `doctl serverless deploy` on the user's machine, including when `--remote-build` is required.

When in doubt — especially if the user might iterate on the function later — ask before scaffolding a `doctl` project for something that could live as inline code.

---

## Supported Tools

### Namespace Tools

- **functions-list-namespaces**
  List all DigitalOcean Functions namespaces. Returns namespace metadata including api_host, region, label, and UUID.
  **Arguments:** None

- **functions-get-namespace**
  Get a DigitalOcean Functions namespace by ID. Returns full namespace details including api_host and key for data plane access.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace

- **functions-create-namespace**
  Create a new DigitalOcean Functions namespace.
  **Arguments:**
    - `Label` (string, required): A human-readable label for the namespace
    - `Region` (string, required): The region slug where the namespace will be created (e.g. nyc1, sfo1)

- **functions-delete-namespace**
  Delete a DigitalOcean Functions namespace. This permanently removes the namespace and all its functions, packages, and triggers.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace to delete

### Access Key Tools

- **functions-list-access-keys**
  List access keys for a DigitalOcean Functions namespace. Returns metadata only (name, id, creation/expiry timestamps) — secret values are NOT returned and cannot be retrieved once a key has been created. Keys whose names start with `mcp-do-` are reserved for this MCP server's own internal use and must not be deleted by agents.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace

- **functions-create-access-key**
  Create an access key for a DigitalOcean Functions namespace. The returned secret appears only in this response and cannot be retrieved later — store it immediately. Typical use: creating a short-lived credential so `doctl serverless connect <namespace> --access-key dof_v1_<id>:<secret>` can connect without a full DigitalOcean API token. For agent-created keys: use the prefix `mcp-agent-` followed by a timestamp (never `mcp-do-`, which is reserved for the MCP server and auto-cleaned); set `ExpiresIn` to `24h` so the key expires on its own — no manual cleanup needed. Requires the `function:admin` scope on the caller's API token.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `Name` (string, required): A name for the access key. For agent-created keys use the prefix `mcp-agent-` followed by a timestamp. Never use the `mcp-do-` prefix — it is reserved for the MCP server.
    - `ExpiresIn` (string, optional): Expiration duration such as `"24h"` or `"7d"` (minimum `"1h"`). Use `"24h"` for agent-created keys. Omit only when the user explicitly asks for a non-expiring key.

- **functions-delete-access-key**
  Delete an access key for a DigitalOcean Functions namespace. Irreversible. Agent-created keys (prefix `mcp-agent-`) expire on their own after 24h, so proactive deletion is not required — only delete one when the user explicitly asks. Do NOT delete keys starting with `mcp-do-` (managed by the MCP server itself). Do NOT delete keys with any other name prefix without explicit user consent.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `KeyID` (string, required): The ID of the access key to delete

---

### Action Tools

- **functions-list-actions**
  List all actions in a DigitalOcean Functions namespace. Returns action metadata including name, namespace, version, and limits.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `Limit` (number, optional): Number of actions to return (0-200, default 30). Use 0 for maximum.
    - `Skip` (number, optional): Number of actions to skip for pagination

- **functions-get-action**
  Get detailed information about a specific action in a DigitalOcean Functions namespace, including its configuration and optionally its source code.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActionName` (string, required): The name of the action
    - `PackageName` (string, optional): The package containing the action, if applicable
    - `IncludeCode` (boolean, optional): Whether to include the action's source code in the response. Default is false.

- **functions-create-or-update-action**
  Create or update an action in a DigitalOcean Functions namespace. If the action already exists it will be overwritten.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActionName` (string, required): The name of the action to create or update
    - `PackageName` (string, optional): The package to place the action in, if applicable
    - `Kind` (string, optional): Runtime kind (e.g. nodejs:20, python:3.11, go:default, php:default, blackbox, sequence)
    - `Code` (string, optional): The source code for the action (when kind is not blackbox)
    - `Image` (string, optional): Container image name (when kind is blackbox)
    - `Main` (string, optional): Main entrypoint of the action code
    - `Components` (array of strings, optional): For sequence actions, the list of action names in order
    - `Timeout` (number, optional): Action timeout in milliseconds (default 60000)
    - `Memory` (number, optional): Action memory in megabytes (default 256)
    - `Logs` (number, optional): Max log size in megabytes (default 10)
    - `Annotations` (array of objects, optional): Key-value annotations for the action
    - `Parameters` (array of objects, optional): Default parameter bindings for the action

- **functions-delete-action**
  Delete an action from a DigitalOcean Functions namespace.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActionName` (string, required): The name of the action to delete
    - `PackageName` (string, optional): The package containing the action, if applicable

- **functions-invoke-action**
  Invoke a function action in a DigitalOcean Functions namespace. By default this is a blocking invocation that waits for the result.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActionName` (string, required): The name of the action to invoke
    - `PackageName` (string, optional): The package containing the action, if applicable
    - `Blocking` (boolean, optional): Whether to wait for the invocation to complete. Default is true.
    - `Result` (boolean, optional): Return only the result of a blocking activation. Default is false.
    - `InvokeTimeout` (number, optional): Max wait time in milliseconds for a blocking response (default/max 60000)
    - `Payload` (object, optional): JSON payload to pass as parameters to the action

---

### Package Tools

- **functions-list-packages**
  List all packages in a DigitalOcean Functions namespace. Packages group related actions together.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `Limit` (number, optional): Number of packages to return (0-200, default 30). Use 0 for maximum.
    - `Skip` (number, optional): Number of packages to skip for pagination
    - `Public` (boolean, optional): Include publicly shared packages in the result

- **functions-get-package**
  Get detailed information about a specific package in a DigitalOcean Functions namespace, including its actions, parameters, and annotations.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `PackageName` (string, required): The name of the package

- **functions-create-or-update-package**
  Create or update a package in a DigitalOcean Functions namespace. Packages are used to group related actions.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `PackageName` (string, required): The name of the package to create or update
    - `Publish` (boolean, optional): Whether to make the package publicly accessible
    - `Annotations` (array of objects, optional): Key-value annotations for the package
    - `Parameters` (array of objects, optional): Default parameter bindings for actions in the package
    - `Binding` (object, optional): Package binding with 'namespace' and 'name' fields to bind to another package

- **functions-delete-package**
  Delete a package from a DigitalOcean Functions namespace.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `PackageName` (string, required): The name of the package to delete
    - `Force` (boolean, optional): Force delete the package even if it contains actions. Default is false.

---

### Trigger Tools

- **functions-list-triggers**
  List all triggers for a DigitalOcean Functions namespace.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace

- **functions-get-trigger**
  Get a specific trigger in a DigitalOcean Functions namespace.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `TriggerName` (string, required): The name of the trigger

- **functions-create-trigger**
  Create a scheduled trigger for a function in a DigitalOcean Functions namespace. Currently only SCHEDULED type triggers are supported.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `Name` (string, required): A name for the trigger
    - `Function` (string, required): The name of the function to invoke
    - `Cron` (string, required): A cron expression defining the schedule (e.g. '*/5 * * * *' for every 5 minutes)
    - `IsEnabled` (boolean, optional): Whether the trigger is enabled. Defaults to true.
    - `Body` (object, optional): Optional JSON payload to pass to the function on each invocation

- **functions-update-trigger**
  Update a trigger in a DigitalOcean Functions namespace. You can enable/disable the trigger or change the cron schedule.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `TriggerName` (string, required): The name of the trigger to update
    - `IsEnabled` (boolean, optional): Whether the trigger should be enabled or disabled
    - `Cron` (string, optional): Updated cron expression for the schedule
    - `Body` (object, optional): Updated JSON payload to pass to the function

- **functions-delete-trigger**
  Delete a trigger from a DigitalOcean Functions namespace.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `TriggerName` (string, required): The name of the trigger to delete

---

### Activation Tools

- **functions-list-activations**
  List activations (invocation records) for a DigitalOcean Functions namespace. Activations record every function invocation with timing, status, and optional response data.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `FunctionName` (string, optional): Filter activations by function name
    - `Limit` (number, optional): Number of activations to return (0-200, default 30). Use 0 for maximum.
    - `Skip` (number, optional): Number of activations to skip for pagination
    - `Since` (number, optional): Only include activations after this timestamp (milliseconds since epoch)
    - `Upto` (number, optional): Only include activations before this timestamp (milliseconds since epoch)
    - `IncludeDocs` (boolean, optional): Include full activation details in the list response

- **functions-get-activation**
  Get the full activation record for a specific function invocation, including response, logs, timing, and status.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActivationID` (string, required): The activation ID

- **functions-get-activation-logs**
  Get only the logs for a specific function activation. Useful for debugging function execution.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActivationID` (string, required): The activation ID

- **functions-get-activation-result**
  Get only the result of a specific function activation. Returns the function's return value and status.
  **Arguments:**
    - `NamespaceID` (string, required): The UUID of the namespace
    - `ActivationID` (string, required): The activation ID

---

### Deployment Guide Tool

- **functions-deployment-guide**
  Return the authoritative step-by-step guide for deploying a DigitalOcean Functions project from a local directory using `doctl serverless deploy`. Call this tool first when the user asks to deploy a function, update a deployed function's code, or set up a new Functions project. The guide covers preflight (`doctl` install + auth + serverless plugin), namespace setup and `doctl serverless connect` (with the access-key fallback), project scaffolding options, the local-vs-remote build decision, the deploy command itself, and post-deploy verification via the other `functions-*` MCP tools. Do not call this tool for routine per-action CRUD — use `functions-create-or-update-action` and related tools directly for those. The returned content is markdown; follow its instructions exactly rather than paraphrasing.
  **Arguments:** None

---

## Example Usage

- **List namespaces:**
  Tool: `functions-list-namespaces`

- **Create a namespace:**
  Tool: `functions-create-namespace`
  Arguments:
    - `Label`: `"my-functions"`
    - `Region`: `"nyc1"`

- **Create an action:**
  Tool: `functions-create-or-update-action`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `ActionName`: `"hello"`
    - `Kind`: `"nodejs:20"`
    - `Code`: `"function main(args) { return { body: 'Hello, ' + (args.name || 'World') + '!' } }"`

- **Invoke an action:**
  Tool: `functions-invoke-action`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `ActionName`: `"hello"`
    - `Payload`: `{"name": "MCP"}`

- **Create a package:**
  Tool: `functions-create-or-update-package`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `PackageName`: `"my-utils"`

- **Create a scheduled trigger:**
  Tool: `functions-create-trigger`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `Name`: `"nightly-job"`
    - `Function`: `"hello"`
    - `Cron`: `"0 0 * * *"`

- **List activations for a function:**
  Tool: `functions-list-activations`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `FunctionName`: `"hello"`
    - `Limit`: `10`

- **Get activation logs:**
  Tool: `functions-get-activation-logs`
  Arguments:
    - `NamespaceID`: `"fn-abc123-..."`
    - `ActivationID`: `"abc123def456..."`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Limit` and `Skip` arguments.
- All responses are returned in JSON format for easy parsing and integration.
- Actions, packages, and activations require an access key to the OpenWhisk data plane. The MCP server manages access key creation and caching automatically — no manual key management is needed.
- Access keys created by the server use a `mcp-do-` prefix and a 24-hour TTL. Orphaned keys from previous sessions are cleaned up automatically on first use.
- There is a limit of 200 access keys per DigitalOcean account. The server's LRU cache and automatic cleanup prevent hitting this limit under normal usage.
