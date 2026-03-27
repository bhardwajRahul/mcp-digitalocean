## DigitalOcean Container Registry (DOCR) Tools

This directory provides tools for managing DigitalOcean Container Registries, repositories, subscriptions, and garbage collection via the MCP Server. All operations are exposed as tools with argument-based input—no resource URIs are used. Pagination and filtering are supported where applicable.

---

## Supported Tools

### Registry Tools

- **docr-get**  
  Get information about a specific container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry

- **docr-list**  
  List all container registries.  
  **Arguments:** None

- **docr-create**  
  Create a new container registry.  
  **Arguments:**
    - `Name` (string, required): Name of the container registry
    - `SubscriptionTierSlug` (string, optional): Subscription tier slug (e.g., 'starter', 'basic', 'professional')
    - `Region` (string, optional): Region slug for the registry (e.g., 'nyc3', 'sfo3')

- **docr-delete**  
  Delete a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry to delete

- **docr-docker-credentials**  
  Get Docker credentials for a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `ReadWrite` (boolean, optional): Whether the credentials should have read-write access (default: false, read-only)
    - `ExpirySeconds` (number, optional): Number of seconds until the credentials expire. If not set, credentials do not expire

- **docr-options**  
  Get available container registry options including subscription tiers and regions.  
  **Arguments:** None

- **docr-validate-name**  
  Check if a container registry name is available.  
  **Arguments:**
    - `Name` (string, required): Name to validate for availability

---

### Repository Tools

- **docr-repository-list**  
  List repositories in a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Page` (number, default: 1): Page number
    - `PerPage` (number, default: 20): Items per page
    - `PageToken` (string, optional): Token for paginating through results

- **docr-repository-tag-list**  
  List tags for a repository in a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Repository` (string, required): Name of the repository
    - `Page` (number, default: 1): Page number
    - `PerPage` (number, default: 20): Items per page

- **docr-repository-tag-delete**  
  Delete a tag from a repository in a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Repository` (string, required): Name of the repository
    - `Tag` (string, required): Tag to delete

- **docr-repository-manifest-list**  
  List manifests for a repository in a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Repository` (string, required): Name of the repository
    - `Page` (number, default: 1): Page number
    - `PerPage` (number, default: 20): Items per page

- **docr-repository-manifest-delete**  
  Delete a manifest from a repository in a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Repository` (string, required): Name of the repository
    - `Digest` (string, required): Digest of the manifest to delete (e.g., 'sha256:abc123...')

---

### Subscription Tools

- **docr-subscription-get**  
  Get the current container registry subscription information.  
  **Arguments:** None

- **docr-subscription-update**  
  Update the container registry subscription tier.  
  **Arguments:**
    - `TierSlug` (string, required): Subscription tier slug to update to (e.g., 'starter', 'basic', 'professional')

---

### Garbage Collection Tools

- **docr-garbage-collection-start**  
  Start a garbage collection for a container registry to free up storage.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Type` (string, optional): Type of garbage collection to perform (e.g., 'untagged manifests and unreferenced blobs' or 'unreferenced blobs only')

- **docr-garbage-collection-get**  
  Get the active garbage collection for a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry

- **docr-garbage-collection-list**  
  List garbage collections for a container registry.  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `Page` (number, default: 1): Page number
    - `PerPage` (number, default: 20): Items per page

- **docr-garbage-collection-update**  
  Update a garbage collection for a container registry (e.g., to cancel it).  
  **Arguments:**
    - `RegistryName` (string, required): Name of the container registry
    - `GarbageCollectionUUID` (string, required): UUID of the garbage collection to update
    - `Cancel` (boolean, required): Set to true to cancel the garbage collection

---

## Example Usage

- **Get a registry:**  
  Tool: `docr-get`  
  Arguments:
    - `RegistryName`: `"my-registry"`

- **Create a registry:**  
  Tool: `docr-create`  
  Arguments:
    - `Name`: `"my-registry"`
    - `SubscriptionTierSlug`: `"basic"`
    - `Region`: `"nyc3"`

- **List repositories:**  
  Tool: `docr-repository-list`  
  Arguments:
    - `RegistryName`: `"my-registry"`
    - `Page`: `1`
    - `PerPage`: `20`

- **List tags for a repository:**  
  Tool: `docr-repository-tag-list`  
  Arguments:
    - `RegistryName`: `"my-registry"`
    - `Repository`: `"my-app"`
    - `Page`: `1`
    - `PerPage`: `20`

- **Delete a tag:**  
  Tool: `docr-repository-tag-delete`  
  Arguments:
    - `RegistryName`: `"my-registry"`
    - `Repository`: `"my-app"`
    - `Tag`: `"v1.0.0"`

- **Get Docker credentials:**  
  Tool: `docr-docker-credentials`  
  Arguments:
    - `RegistryName`: `"my-registry"`
    - `ReadWrite`: `true`
    - `ExpirySeconds`: `3600`

- **Start garbage collection:**  
  Tool: `docr-garbage-collection-start`  
  Arguments:
    - `RegistryName`: `"my-registry"`

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported for list endpoints via `Page` and `PerPage` arguments.
- Repository list endpoints also support `PageToken` for cursor-based pagination.
- All responses are returned in JSON format for easy parsing and integration.
- For endpoints that require a registry name or repository name, provide the appropriate value in your query.
- Docker credentials can be configured with read-write access and expiration time for enhanced security.
- Garbage collection helps manage storage by removing unused images and manifests from your registry.
