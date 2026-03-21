# Documentation Tools

Read-only tools for querying DigitalOcean's public documentation. No authentication required.

## Tools

| Tool | Description |
|------|-------------|
| `docs-search` | Full-text search across all DigitalOcean documentation |
| `docs-get-page` | Fetch the full markdown content of a specific docs page |
| `docs-find-for-service` | List documentation pages for a given DigitalOcean service |
| `docs-get-quickstart` | Get the quickstart guide for a service |

## How It Works

- Indexes `docs.digitalocean.com/llms.txt` and per-service `llms.txt` files
- Fetches raw markdown via the `index.html.md` endpoint
- In-memory caching (30 min for pages, 1 hour for indexes)
- Supports common service name aliases (e.g., "k8s" → "kubernetes", "gpu" → "bare-metal-gpus")

## Examples

### Search Documentation

Search for documentation about Kubernetes networking:

```
Tool: docs-search
Arguments: { "Query": "kubernetes networking", "Limit": 5 }
```

### Get a Specific Page

Fetch the Droplets quickstart guide:

```
Tool: docs-get-page
Arguments: { "URL": "/products/droplets/getting-started/quickstart/" }
```

### Browse a Service

List all documentation for App Platform:

```
Tool: docs-find-for-service
Arguments: { "Service": "app platform" }
```

### Get a Quickstart Guide

Get the getting-started guide for databases:

```
Tool: docs-get-quickstart
Arguments: { "Service": "databases" }
```
