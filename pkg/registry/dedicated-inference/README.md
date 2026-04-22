# Dedicated Inference (`dedicated-inference`)

## What this service does

This package registers **Dedicated Inference** tools for the DigitalOcean MCP server. Dedicated Inference lets you run model workloads on GPU-backed infrastructure in your account. The tools call the **public** DigitalOcean API (`/v2/dedicated-inferences`) through `[godo](https://github.com/digitalocean/godo)`.

**Enable these tools** by including the service key when you configure the server (see the main project docs): service name `**dedicated-inference`**.

**Code layout**


| Path                                                                     | Purpose                                                                                              |
| ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| `[dedicated_inference_tools.go](dedicated_inference_tools.go)`           | Tool handlers and MCP tool definitions                                                               |
| `[dedicated_inference_tools_test.go](dedicated_inference_tools_test.go)` | Unit tests (mocked API client)                                                                       |
| `[spec/](spec/)`                                                         | Optional JSON Schema snippets describing HTTP request bodies (reference only; not loaded at runtime) |


---

## Tools exposed


| Tool                         | What it does                                                                                                                                       |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `dedicated-inference-create` | Create a new Dedicated Inference instance. Returns the resource and an optional initial auth `token`.                                              |
| `dedicated-inference-get`    | Fetch one instance by ID.                                                                                                                          |
| `dedicated-inference-list`   | List instances with optional `Region`, `Name`, `Page`, and `PerPage`. Result is a **JSON array** only (no pagination metadata in the tool output). |
| `dedicated-inference-update` | Update an instance’s deployment spec and/or Hugging Face secret.                                                                                   |
| `dedicated-inference-delete` | Delete an instance by ID.                                                                                                                          |


---

## Arguments (summary)

### `dedicated-inference-create`


| Argument               | Required | Description                                                                                                                                       |
| ---------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Name`                 | yes      | Instance name                                                                                                                                     |
| `Region`               | yes      | Region slug (e.g. `nyc2`)                                                                                                                         |
| `ModelDeployments`     | yes      | Non-empty list of deployments (`ModelSlug`, `ModelProvider`, optional `ModelID`, `Accelerators` with `AcceleratorSlug`, `Scale`, optional `Type`) |
| `EnablePublicEndpoint` | no       | Public endpoint toggle                                                                                                                            |
| `VPCUUID`              | no       | VPC UUID; omit to use platform defaults                                                                                                           |
| `HuggingFaceToken`     | no       | Write-only; not returned in responses                                                                                                             |


The handler sets `spec.version` to `1` on create.

### `dedicated-inference-update`


| Argument               | Required | Description                                                                                                                                                                                                                                                                             |
| ---------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DedicatedInferenceID` | yes      | Instance UUID                                                                                                                                                                                                                                                                           |
| Other fields           | no       | Same family as create (`Name`, `Region`, `EnablePublicEndpoint`, `VPCUUID`, `ModelDeployments`, `HuggingFaceToken`)—only values you pass are forwarded into the API request. For PATCH semantics and required fields, rely on the **public API** and the behavior of your request body. |
| `HuggingFaceToken`     | no       | Omit to leave existing secret; set to replace                                                                                                                                                                                                                                           |


### `dedicated-inference-get` / `dedicated-inference-delete`

- **get:** `DedicatedInferenceID` (required).
- **delete:** `DedicatedInferenceID` (required). Returns a small JSON success payload.

### `dedicated-inference-list`

- Optional: `Region`, `Name`, `Page`, `PerPage`.

---

## Reference schemas (`spec/*.json`)


| File                                                                                         | Contents                                                                         |
| -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `[spec/dedicated-inference-create-schema.json](spec/dedicated-inference-create-schema.json)` | HTTP-style `spec` + optional `secrets` (snake_case)                              |
| `[spec/dedicated-inference-update-schema.json](spec/dedicated-inference-update-schema.json)` | Update body shape; MCP passes `DedicatedInferenceID` as a separate tool argument |


---

## Secrets

- `HuggingFaceToken` is **write-only** in outputs.
- On update, omitting it keeps the previous secret; setting it replaces the value.

---

## Responses and polling

- **Create** returns `dedicated_inference` plus optional `token`(API-issued auth token, if returned).
- **Get / update** return the dedicated inference resource JSON.
- Operations may complete asynchronously; use **get** to poll until status stabilizes.

---

## Auth

Callers need a DigitalOcean API token with the appropriate `**dedicated_inference`** scopes for the operations they use.