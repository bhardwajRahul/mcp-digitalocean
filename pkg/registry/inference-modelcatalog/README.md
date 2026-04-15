## Model Catalog MCP Tools

This directory contains tools and prompts for managing DigitalOcean Model Catalog via the MCP Server. The Model Catalog provides access to AI models available on DigitalOcean's Inference platform.

---

## Tools

- **inference-model-catalog-search**  
  Search for models in the catalog. Returns model UUIDs matching the search criteria.  
  **Arguments:** `SearchQuery` (optional) - Search string. Empty returns all models.

- **inference-model-catalog-get-card**  
  Get detailed metadata for a specific model.  
  **Arguments:** `ModelUUID` (required) - The model's UUID.

## Prompts

- **model-comparison**  
  Compare two models side-by-side with detailed metrics (pricing, parameters, capabilities, performance).  
  **Arguments:** `ModelUUID1`, `ModelUUID2` (both required)

- **search-by-task**  
  Find models matching a task with optional constraints (provider, deployment type, pricing, context window).  
  **Arguments:**  
  - `Task` (optional) - Task description (e.g., "low-latency chat")
  - `Provider` (optional) - Filter by provider (e.g., "OpenAI", "Meta")
  - `DeploymentType` (optional) - "serverless" or "dedicated"
  - `MinContextWindow` (optional) - Minimum tokens (e.g., "100000")
  - `MaxInputPrice` (optional) - Max input price per million tokens (e.g., "5.0")
  - `MaxOutputPrice` (optional) - Max output price per million tokens (e.g., "15.0")

---

## Example Usage

**Search models:**
```
Tool: inference-model-catalog-search
Args: {"SearchQuery": "llama"}
```

**Get model details:**
```
Tool: inference-model-catalog-get-card
Args: {"ModelUUID": "12345678-1234-1234-1234-123456789012"}
```

**Compare two models:**
```
Prompt: model-comparison
Args: {
  "ModelUUID1": "uuid-1",
  "ModelUUID2": "uuid-2"
}
```

**Find models for a task:**
```
Prompt: search-by-task
Args: {
  "Task": "low-latency chat",
  "Provider": "meta",
  "MaxInputPrice": "5.0"
}
```

---

## Notes

- Empty `SearchQuery` returns all available models
- All price values are per million tokens
- Context window values can be specified with or without "K" suffix (e.g., "128K" or "128000")
- Provider and deployment type matching is case-insensitive
- A valid DigitalOcean API token is required
