## Documentation Tools

Read-only tools for querying DigitalOcean's public documentation. No authentication required.

---

## Supported Tools

- **docs-search**
  Full-text search across all DigitalOcean documentation.
  **Arguments:**
    - `Query` (string, required): Search query string
    - `Limit` (number, default: 10): Maximum number of results to return

- **docs-get-page**
  Fetch the full markdown content of a specific docs page.
  **Arguments:**
    - `URL` (string, required): Full URL or path of the docs page (e.g., `https://docs.digitalocean.com/products/droplets/getting-started/quickstart/` or `/products/droplets/getting-started/quickstart/`)

- **docs-find-for-service**
  List documentation pages for a given DigitalOcean service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`, `"databases"`)

- **docs-get-quickstart**
  Get the quickstart or getting-started guide for a service.
  **Arguments:**
    - `Service` (string, required): DigitalOcean service name (e.g., `"droplets"`, `"kubernetes"`, `"app platform"`)

---

## How It Works

- Indexes `docs.digitalocean.com/llms.txt` and per-service `llms.txt` files
- Fetches raw markdown via the `index.html.md` endpoint
- In-memory caching (30 min for pages, 1 hour for indexes)
- Supports common service name aliases (e.g., "k8s" → "kubernetes", "gpu" → "bare-metal-gpus")

---

## Example Usage

- **Search documentation:**
  Tool: `docs-search`
  Arguments:
    - `Query`: `"kubernetes networking"`
    - `Limit`: `5`

- **Fetch a specific page:**
  Tool: `docs-get-page`
  Arguments:
    - `URL`: `"/products/droplets/getting-started/quickstart/"`

- **Browse a service:**
  Tool: `docs-find-for-service`
  Arguments:
    - `Service`: `"app platform"`

- **Get a quickstart guide:**
  Tool: `docs-get-quickstart`
  Arguments:
    - `Service`: `"databases"`

---

## Notes

- All tools are read-only and do not require a DigitalOcean API token.
- All tools use argument-based input; do not use resource URIs.
- All responses are returned as markdown text.
- Service name aliases are supported (e.g., "k8s" for "kubernetes", "postgres" for "postgresql").
