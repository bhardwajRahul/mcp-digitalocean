## Task: Generate /llms-index.json as a Hugo output format

We want a static JSON file at /llms-index.json that contains structured
metadata for every docs page. This lets AI agents search and filter our
docs programmatically without needing a search API. It must be generated
at build time by Hugo so it auto-updates on every deploy.

### What to do

1. Add a new custom output format in the Hugo config for JSON. This is the
   same pattern used for the existing llms.txt and llms-full.txt outputs.
   The output format should produce a single file at the site root:
   /llms-index.json

2. Create a layout template that iterates over all regular pages and emits
   a JSON array of objects. Each object should have:

   ```json
   {
     "title": "How to Resize Droplets for Vertical Scaling",
     "description": "Resize a Droplet to change the amount of CPU and RAM...",
     "product": "Droplets",
     "section": "how-to",
     "url": "https://docs.digitalocean.com/products/droplets/how-to/resize/",
     "markdown_url": "https://docs.digitalocean.com/products/droplets/how-to/resize/index.html.md",
     "tags": ["resize", "scaling", "vertical-scaling"]
   }
   ```

   Field notes:
   - product: Same derivation as the frontmatter task — top-level section
     under /products/, title-cased. For non-product pages use the section
     name (e.g. "Platform", "Support", "Reference").
   - section: The content type/category within the product — typically
     "how-to", "getting-started", "details", "concepts", "reference",
     or "support". Derive from the URL path.
   - description: Use .Description or .Params.description. If empty,
     use .Summary truncated to 200 chars.
   - tags: Use .Params.tags if defined in frontmatter, otherwise omit
     the field entirely (don't emit an empty array).
   - markdown_url: The .html.md equivalent of the page URL.

3. The JSON must be valid — properly escaped strings, no trailing commas.
   Use Hugo's jsonify function or build the JSON with a template that
   handles escaping correctly. Test with `jq . < llms-index.json` after
   building.

4. Add a reference to this file in two places:

   a. At the top of llms.txt, in the "For AI agents" blockquote, add:
      `> **Structured index (JSON):** https://docs.digitalocean.com/llms-index.json`

   b. In robots.txt, add:
      `Llms-index: https://docs.digitalocean.com/llms-index.json`

### Size management

The file will likely be 500KB-1MB for the full site. That's fine — it's a
single fetch that agents can cache and filter client-side. Do NOT paginate
or split it. The whole point is that it's one file.

### Verification

1. Run `hugo build` and confirm /llms-index.json exists in the output
2. Validate with: `jq . public/llms-index.json > /dev/null && echo "valid"`
3. Check the count: `jq length public/llms-index.json` (should match your
   total page count roughly)
4. Spot-check a few entries for correct product, section, and URL values
5. Confirm llms.txt and robots.txt contain the new references
6. Confirm existing outputs (llms.txt, llms-full-*.txt, .html.md pages)
   are unchanged
