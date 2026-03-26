# DigitalOcean Serverless Functions Tools

This directory provides tool-based handlers for interacting with DigitalOcean Serverless Functions via the MCP
Server. All operations are exposed as tools that accept structured arguments. The serverless service covers
namespace and trigger management.

## Supported Tools

### Namespace

- **serverless-namespace-list**
    - List all serverless function namespaces.
    - Arguments: none

- **serverless-namespace-get**
    - Get a serverless function namespace by ID.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).

- **serverless-namespace-create**
    - Create a new serverless function namespace.
    - Arguments:
        - `Label` (string, required): Label for the namespace.
        - `Region` (string, required): Region slug for the namespace (e.g., 'nyc1', 'sfo3').

- **serverless-namespace-delete**
    - Delete a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).

### Trigger

- **serverless-trigger-list**
    - List all triggers for a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).

- **serverless-trigger-get**
    - Get a trigger by name for a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `TriggerName` (string, required): Name of the trigger.

- **serverless-trigger-create**
    - Create a new trigger for a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `Name` (string, required): Name of the trigger.
        - `Function` (string, required): Function to invoke (e.g., 'mypackage/myfunction').
        - `Type` (string, required): Type of trigger (e.g., 'SCHEDULED').
        - `IsEnabled` (boolean): Whether the trigger is enabled (default: false).
        - `Cron` (string): Cron expression for scheduled triggers (e.g., '*/5 * * * *').
        - `Body` (object): JSON body to pass to the function when triggered.

- **serverless-trigger-update**
    - Update a trigger for a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `TriggerName` (string, required): Name of the trigger to update.
        - `IsEnabled` (boolean): Whether the trigger is enabled.
        - `Cron` (string): Updated cron expression for scheduled triggers.
        - `Body` (object): Updated JSON body to pass to the function when triggered.

- **serverless-trigger-delete**
    - Delete a trigger from a serverless function namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `TriggerName` (string, required): Name of the trigger to delete.

### Function

These tools interact directly with the OpenWhisk API hosted at each namespace's `api_host`.
They first resolve the namespace to obtain the API host and auth key, then call the OpenWhisk REST API.

- **serverless-function-list**
    - List all functions in a serverless namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `Limit` (number): Maximum number of functions to return (default: 100).
        - `Skip` (number): Number of functions to skip for pagination (default: 0).

- **serverless-function-get**
    - Get details of a specific function in a serverless namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `FunctionName` (string, required): Name of the function (e.g., 'mypackage/myfunction' or 'myfunction').

- **serverless-function-create**
    - Create or update a serverless function with inline code.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `FunctionName` (string, required): Name of the function (e.g., 'mypackage/myfunction' or 'myfunction').
        - `Kind` (string, required): Runtime kind (e.g., 'nodejs:18', 'python:3.11', 'go:1.21', 'php:8.2').
        - `Code` (string, required): Source code of the function.
        - `WebExport` (boolean): Make the function publicly accessible as a web action (default: false).
        - `Overwrite` (boolean): Overwrite the function if it already exists (default: true).

- **serverless-function-delete**
    - Delete a function from a serverless namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `FunctionName` (string, required): Name of the function to delete.

- **serverless-function-invoke**
    - Invoke a serverless function and return the result.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `FunctionName` (string, required): Name of the function to invoke.
        - `Params` (object): JSON parameters to pass to the function.
        - `Blocking` (boolean): Wait for the function to complete before returning (default: true).

### Activation

These tools retrieve activation (invocation) records and logs from the OpenWhisk API.

- **serverless-activation-list**
    - List activation records for a serverless namespace.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `FunctionName` (string): Filter activations by function name.
        - `Limit` (number): Maximum number of activations to return (default: 100).
        - `Skip` (number): Number of activations to skip for pagination (default: 0).
        - `Since` (number): Only return activations after this timestamp (epoch milliseconds).
        - `Upto` (number): Only return activations before this timestamp (epoch milliseconds).

- **serverless-activation-get**
    - Get details of a specific activation record.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `ActivationID` (string, required): ID of the activation.

- **serverless-activation-logs**
    - Get logs for a specific function activation.
    - Arguments:
        - `NamespaceID` (string, required): The `namespace` field from the namespace object (e.g., `fn-abc123-...`).
        - `ActivationID` (string, required): ID of the activation.

---

## Example Usage

- List all namespaces:
    - Tool: `serverless-namespace-list`
    - Arguments: `{}`

- Get namespace details:
    - Tool: `serverless-namespace-get`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab" }`

- Create a new namespace:
    - Tool: `serverless-namespace-create`
    - Arguments: `{ "Label": "my-functions", "Region": "nyc1" }`

- Delete a namespace:
    - Tool: `serverless-namespace-delete`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab" }`

- List triggers for a namespace:
    - Tool: `serverless-trigger-list`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab" }`

- Get a specific trigger:
    - Tool: `serverless-trigger-get`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "TriggerName": "my-cron-trigger" }`

- Create a scheduled trigger:
    - Tool: `serverless-trigger-create`
    - Arguments:
      ```json
      {
        "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "Name": "my-cron-trigger",
        "Function": "mypackage/myfunction",
        "Type": "SCHEDULED",
        "IsEnabled": true,
        "Cron": "*/5 * * * *",
        "Body": { "key": "value" }
      }
      ```

- Update a trigger (disable it):
    - Tool: `serverless-trigger-update`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "TriggerName": "my-cron-trigger", "IsEnabled": false }`

- Delete a trigger:
    - Tool: `serverless-trigger-delete`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "TriggerName": "my-cron-trigger" }`

- List all functions in a namespace:
    - Tool: `serverless-function-list`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab" }`

- Create a Node.js function:
    - Tool: `serverless-function-create`
    - Arguments:
      ```json
      {
        "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab",
        "FunctionName": "hello",
        "Kind": "nodejs:18",
        "Code": "function main(params) { return { body: 'Hello ' + (params.name || 'World') }; }",
        "WebExport": true
      }
      ```

- Invoke a function with parameters:
    - Tool: `serverless-function-invoke`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "FunctionName": "hello", "Params": { "name": "John" } }`

- Delete a function:
    - Tool: `serverless-function-delete`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "FunctionName": "hello" }`

- List recent activations:
    - Tool: `serverless-activation-list`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab" }`

- List activations for a specific function:
    - Tool: `serverless-activation-list`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "FunctionName": "hello", "Limit": 10 }`

- Get activation details:
    - Tool: `serverless-activation-get`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "ActivationID": "abc123def456" }`

- Get activation logs:
    - Tool: `serverless-activation-logs`
    - Arguments: `{ "NamespaceID": "fn-a1b2c3d4-5678-90ab-cdef-1234567890ab", "ActivationID": "abc123def456" }`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- All responses are returned as JSON-formatted text.
- Error handling is consistent: errors are returned in the tool result with an error flag and message.
