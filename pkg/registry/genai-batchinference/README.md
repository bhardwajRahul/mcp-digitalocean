# GenAI Batch Inference (`genai-batchinference`)

## What this service does

This package registers **Batch Inference** tools for the DigitalOcean MCP server. Batch Inference allows you to submit large batches of inference requests (JSONL) for asynchronous processing via OpenAI-compatible or Anthropic models. The tools call the DigitalOcean inference proxy at `inference.do-ai.run` through [`godo`](https://github.com/digitalocean/godo).

**Enable these tools** by including the service key when you configure the server: service name **`genai-batchinference`**.

**Code layout**

| Path | Purpose |
| --- | --- |
| [`batch_inference_tools.go`](batch_inference_tools.go) | Tool handlers and MCP tool definitions |
| [`batch_inference_tools_test.go`](batch_inference_tools_test.go) | Unit tests (mocked API client) |
| [`generate.go`](generate.go) | mockgen directive |
| [`mocks.go`](mocks.go) | Generated mock for `godo.BatchInferenceService` |

---

## Tools exposed

| Tool | What it does |
| --- | --- |
| `genai-batch-inference-create-file` | Create a presigned URL for uploading a JSONL input file. |
| `genai-batch-inference-upload-file` | Upload JSONL content to the presigned URL returned by `create-file`. |
| `genai-batch-inference-create` | Create a new batch inference job (OpenAI or Anthropic). |
| `genai-batch-inference-get` | Get a batch inference job's status and metadata. |
| `genai-batch-inference-get-results` | Get the presigned download URL for completed job results. |
| `genai-batch-inference-cancel` | Request cancellation of a running batch inference job. |
| `genai-batch-inference-list` | List batch inference jobs with cursor-based pagination. |

---

## Arguments (summary)

### `genai-batch-inference-create-file`

| Argument | Required | Description |
| --- | --- | --- |
| `FileName` | yes | Name of the JSONL file (must end in `.jsonl`) |

### `genai-batch-inference-upload-file`

| Argument | Required | Description |
| --- | --- | --- |
| `UploadURL` | yes | Presigned upload URL from `create-file` response |
| `Content` | yes | JSONL content to upload (newline-delimited JSON) |

### `genai-batch-inference-create`

| Argument | Required | Description |
| --- | --- | --- |
| `Provider` | yes | `openai` or `anthropic` |
| `FileID` | yes | UUID of a previously uploaded `.jsonl` file |
| `CompletionWindow` | yes | e.g. `24h` |
| `RequestID` | no | Client-supplied idempotency key |
| `Endpoint` | no | OpenAI batch API path (required for OpenAI, e.g. `/v1/chat/completions`) |

### `genai-batch-inference-get` / `genai-batch-inference-get-results` / `genai-batch-inference-cancel`

| Argument | Required | Description |
| --- | --- | --- |
| `BatchID` | yes | UUID of the batch inference job |

### `genai-batch-inference-list`

| Argument | Required | Description |
| --- | --- | --- |
| `Status` | no | Filter by status (e.g. `completed`, `in_progress`) |
| `Limit` | no | Max jobs per page |
| `After` | no | Cursor from previous page's `endCursor` |

---

## Responses

- **create-file**: Returns `file_id`, `upload_url`, and `expires_at`.
- **upload-file**: Returns a success message on successful upload.
- **create**: Returns batch object with `batch_id`, `status`, `provider`, `request_counts`, timestamps.
- **get**: Returns batch object (same shape as create response).
- **get-results**: Returns `output_file_id` and nested `download` with `presigned_url` and `expires_at`.
- **cancel**: Returns the batch object with updated status.
- **list**: Returns Relay-style `edges` (each with `node` and `cursor`) and `page_info` (`hasNextPage`, `endCursor`).

---

## Auth

Callers need a DigitalOcean API token with access to GenAI Batch Inference endpoints. The feature may require enablement on your account.
