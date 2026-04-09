## Task: Include YAML frontmatter in .html.md page output

We serve markdown versions of every docs page at `index.html.md`. Currently
these start with an AI-agent blockquote and then the page content. We want
to prepend a YAML frontmatter block so AI agents can parse structured metadata.

### What to do

1. Find the Hugo output format and layout template that generates `.html.md`
   pages. This is likely defined in our Hugo config (hugo.toml / config.toml)
   as a custom output format, with a corresponding template in layouts/.

2. Modify the template to emit a YAML frontmatter block at the top of each
   .html.md page, BEFORE the existing "For AI agents" blockquote. The
   frontmatter must be fenced with `---` on its own lines.

3. Include these fields in the frontmatter, using Hugo template variables:

   - title: {{ .Title }}
   - description: {{ .Description }} or {{ .Params.description }}
   - product: Derive from the section. For a page at /products/droplets/...,
     the product should be "Droplets". Use the top-level section under
     /products/ and title-case it. For non-product pages (platform, reference,
     support), use that section name instead.
   - url: The canonical URL ({{ .Permalink }})
   - last_updated: {{ .Lastmod }} or {{ .Date }}, formatted as YYYY-MM-DD
   - Only include fields that have values. Skip empty/nil fields.

4. Do NOT change anything else about the .html.md output — the AI agent
   blockquote, page content, and all links should remain exactly as they are.

### Example output

For /products/droplets/how-to/resize/index.html.md, the output should be:

```yaml
---
title: "How to Resize Droplets for Vertical Scaling"
description: "Resize a Droplet to change the amount of CPU and RAM..."
product: "Droplets"
url: "https://docs.digitalocean.com/products/droplets/how-to/resize/"
last_updated: "2026-03-15"
---
```

Followed by the existing content unchanged:

```
> **For AI agents:** The documentation index is at...

# How to Resize Droplets for Vertical Scaling
[...rest of existing content unchanged...]
```

### Verification

Build the site locally and check 3-4 .html.md pages across different sections:
- A product page (e.g. /products/droplets/how-to/create/index.html.md)
- A platform page (e.g. /platform/billing/index.html.md)
- A support page (e.g. /support/why-is-smtp-blocked/index.html.md)

Confirm each has correct frontmatter and the rest of the content is unchanged.
