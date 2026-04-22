# DigitalOcean Functions — Agent Deploy Spec

**Audience:** this document is written for an LLM agent that has access to (a) the `mcp-digitalocean` MCP server's `functions-*` tools, (b) a shell execution tool on the user's machine, and (c) a file-writing tool on the user's machine. It is the single source of truth for how an agent should deploy a DigitalOcean Function.

**Do not paraphrase this guide to the user.** Follow the steps exactly. Where the guide says "stop and ask the user," stop and ask — do not guess.

All commands and flags are verified against `doctl` 1.154.0 (see Appendix D).

---

## 0. When to use this guide

Follow this guide when the user asks to:

- Deploy a function, package, or project to DigitalOcean Functions
- Update an existing deployed function with new code
- Set up a new Functions project from scratch

Do **not** follow this guide for:

- Creating a single action from inline code with no dependencies — use the `functions-create-or-update-action` MCP tool directly.
- Invoking, listing, or inspecting already-deployed functions — use the relevant `functions-*` MCP tools.
- Managing triggers on already-deployed functions — use `functions-create-trigger` / `functions-update-trigger`.

---

## 1. High-level flow

Execute these phases in order. Do not skip phases. If any phase fails, stop and report the exact error before proceeding.

1. **Preflight** — verify `doctl` and the serverless plugin are installed and authenticated.
2. **Namespace setup** — select or create a Functions namespace, then connect `doctl` to it.
3. **Project preparation** — either scaffold a new project on disk or validate an existing one.
4. **Deploy** — run `doctl serverless deploy` and parse the JSON result.
5. **Verify** — invoke the deployed function and inspect the activation via MCP tools.

---

## 2. Phase 1 — Preflight

### 2.1 Check `doctl` is installed

Run:

```bash
doctl version
```

- **If the command succeeds:** note the version and continue.
- **If the command is not found:** install `doctl` using the platform-appropriate command, then re-check:
  - macOS: `brew install doctl`
  - Linux (Debian/Ubuntu): download the latest release tarball from `https://github.com/digitalocean/doctl/releases`, extract, and place `doctl` on `PATH`.
  - Windows: `scoop install doctl` or download the release zip.
- **If installation requires sudo or user consent:** stop and ask the user to run the install command themselves. Do not run `sudo` without permission.

### 2.2 Check `doctl` is authenticated

Run:

```bash
doctl auth list
```

- **If at least one authenticated context is listed:** continue.
- **If no contexts are listed:** stop and tell the user:
  > "I need you to authenticate `doctl` once before I can deploy. Run `doctl auth init` and paste your DigitalOcean API token when prompted. The token must have the `function:admin` scope for `connect` and `deploy` to work. Let me know when that's done."
  Do not attempt to provide a token yourself. Do not proceed until the user confirms.

### 2.3 Check the serverless plugin is installed

Run:

```bash
doctl serverless status
```

- **If the command succeeds and shows a connected namespace:** continue to Phase 2.3 (you are already connected; you may still want to switch namespaces).
- **If the command reports the plugin is not installed:** run:
  ```bash
  doctl serverless install
  ```
  Then re-run `doctl serverless status`.
- **If `doctl serverless status` reports "not connected":** continue to Phase 2.

---

## 3. Phase 2 — Namespace setup

### 3.1 List existing namespaces

Call the MCP tool `functions-list-namespaces`.

- **If zero namespaces exist:** go to 3.2 (create one).
- **If exactly one namespace exists:** use it. Go to 3.3.
- **If multiple namespaces exist:** stop and ask the user which namespace to deploy to. Present them as a short list (label, region, UUID). Do not guess.

### 3.2 Create a namespace (only if none exist)

Stop and ask the user:

> "You don't have any Functions namespaces yet. I'll create one. Which region should I use? (Common choices: `nyc1`, `sfo3`, `fra1`, `sgp1`, `ams3`, `blr1`, `syd1`, `tor1`.) And what label should I use?"

Then call `functions-create-namespace` with the user's answers. Capture the returned `NamespaceID`.

### 3.3 Connect `doctl` to the namespace

Prefer the simplest path that works:

**Option A — use existing `doctl` auth (preferred):** run:

```bash
doctl serverless connect <hint>
```

where `<hint>` is a complete or partial match to the namespace's **label or UUID** (from `functions-list-namespaces`). Tips:

- If the user has exactly one namespace, you can run `doctl serverless connect` with **no argument** — it auto-connects.
- If the hint matches multiple namespaces, `doctl` shows an interactive picker, which you cannot answer programmatically. Always pass a hint unique enough to identify one namespace.
- The API token in the `doctl` context must have the `function:admin` scope.

Then continue:

- **If this succeeds:** go to 3.4.
- **If this fails with an authentication or scope error:** fall through to Option B.

**Option B — connect via an access key:** use this only when Option A fails.

1. Call `functions-list-access-keys` for the namespace. If any keys exist with names matching the prefix `mcp-agent-`, call `functions-delete-access-key` for each. This prevents accumulating stale keys.

   **Prefix rules — strict:**
   - `mcp-agent-*` — safe to delete; these are prior agent-created keys.
   - `mcp-do-*` — **reserved for this MCP server's own internal use.** Never create or delete keys with this prefix. The server manages them automatically (creates on demand, cleans up orphans, TTL-driven expiry). Any `mcp-do-*` key you create will be deleted on the server's next resolution of that namespace.
   - Any other prefix — do **not** delete without explicit user consent; these belong to the user or to other tooling.

2. Call `functions-create-access-key` with:
   - `Name`: `mcp-agent-<YYYYMMDD-HHMMSS>` (e.g. `mcp-agent-20260421-143000`)
   - `ExpiresIn`: `"24h"`

3. Capture the returned key id and secret. **The secret is returned only once and cannot be retrieved later.**
4. Run:
   ```bash
   doctl serverless connect <hint> --access-key dof_v1_<access-key-id>:<secret>
   ```
   The value of `--access-key` is a single string: the literal prefix `dof_v1_`, followed by the access key id, followed by `:`, followed by the secret. Example: `dof_v1_abc123:xyz789`.
5. **Never print the secret to the user in plain form.** Redact it in any summary you produce. Do not write it to a file unless necessary; if you must, write it to a gitignored temp path and delete it immediately after connect succeeds.

### 3.4 Verify the connection

Run:

```bash
doctl serverless status
```

Confirm the output shows the intended namespace as connected. If not, stop and report the mismatch.

---

## 4. Phase 3 — Project preparation

### 4.1 Decide the project shape

Determine which scenario applies:

| Scenario | Condition | Action |
|---|---|---|
| A. Existing project | User pointed at a directory that already contains `project.yml` | Go to 4.5 (validate) |
| B. Single file, no deps | User wants one function, no external dependencies | Scaffold minimal project (4.2) |
| C. Single file, has deps | User wants one function that imports a third-party library | Scaffold project with deps (4.3) |
| D. Multi-function | User wants multiple functions, or one function with multiple source files | Scaffold full project (4.4) |

**Preferred scaffolding method for B/C/D:** use `doctl serverless init` to create a starter project, then customize it:

```bash
doctl serverless init <project-path>
# or, for a specific runtime:
doctl serverless init <project-path> --language <lang>
```

This produces a valid `project.yml` and `packages/` layout automatically, which is more reliable than hand-writing the files. After running `init`, modify the generated files to match the user's code. Only fall back to hand-writing the layout (as described in 4.2–4.4) if `init` is unavailable or produces something the user doesn't want.

### 4.2 Scaffold — single file, no dependencies (Scenario B)

Create exactly this layout at `<project-path>`:

```
<project-path>/
  project.yml
  packages/
    default/
      <function-name>.<ext>
```

Where `<ext>` matches the runtime (`.js`, `.py`, `.go`, `.php`). Example `project.yml`:

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: <function-name>
        runtime: 'nodejs:20'
        web: true
```

Write the user's code to `packages/default/<function-name>.<ext>`.

### 4.3 Scaffold — single file with dependencies (Scenario C)

Use a directory for the function so the deployer can run a dependency install. Layout:

```
<project-path>/
  project.yml
  packages/
    default/
      <function-name>/
        <main-file>.<ext>     # e.g. __main__.py, index.js, main.go
        <deps-file>           # package.json | requirements.txt | go.mod | composer.json
        build.sh              # REQUIRED for Python and PHP; optional for Node.js; not needed for Go
```

**CRITICAL — runtime-specific build triggering.** The deployer (`functions-deployer/src/finder-builder.ts`) only executes a dependency install when it finds a recognized build trigger in the function directory. The dominance order is:

1. `build.sh` (or `build.cmd` on Windows) — runs the script verbatim. Works for all runtimes.
2. `package.json` — special-cased Node.js builder; runs `npm install` automatically.
3. `.include` — use as-is, no build.
4. None of the above — **passthrough zip** (source files are uploaded without any install).

Implications per runtime:

- **Python:** `requirements.txt` alone is **NOT a build trigger**. Without a `build.sh`, the deployer zips `__main__.py` and `requirements.txt` as plain files and `pip install` never runs. At cold-start, `from <third_party> import ...` raises `ModuleNotFoundError` and the invocation returns HTTP 502 "The function did not initialize properly." **You must create a `build.sh`.** Template:
  ```bash
  #!/bin/bash
  set -e
  virtualenv virtualenv
  source virtualenv/bin/activate
  pip install -r requirements.txt
  deactivate
  ```
  Mark it executable: `chmod +x build.sh`.

- **PHP:** `composer.json` alone is **NOT a build trigger**. Same silent-passthrough failure mode as Python. **You must create a `build.sh`.** Template:
  ```bash
  #!/bin/bash
  set -e
  composer install
  ```
  Mark it executable: `chmod +x build.sh`.

- **Node.js:** `package.json` **is** a build trigger. `npm install` runs automatically. A `build.sh` is optional and only needed if the user has a custom build step (e.g. TypeScript compile, bundling). If you do write one, it **replaces** the automatic `npm install` — your script must run `npm install` itself.

- **Go:** neither `build.sh` nor `--remote-build` is required. Presence of `go.mod` with source files triggers the Go remote-build path automatically. A custom `build.sh` is only needed if the user has unusual build requirements.

Deployment of this scenario **should use remote build** (`--remote-build`) unless the user's machine is known to have the correct toolchain (Node, Python, Go, PHP, plus any native-dep system libraries). Remote build is the safer default because it runs the build inside the target runtime container. See Section 5.1 for what `--remote-build` actually does.

### 4.4 Scaffold — multi-function project (Scenario D)

```
<project-path>/
  project.yml
  packages/
    <pkg-name>/
      <func-1>/
        <main-file>.<ext>
        <deps-file>
      <func-2>.<ext>
```

Each subdirectory under `packages/<pkg-name>/` is a function with its own build. Each flat file under `packages/<pkg-name>/` is a single-file function.

### 4.5 Validate an existing project (Scenario A)

Before running deploy, confirm:

1. `project.yml` exists at the project root.
2. A `packages/` directory exists.
3. At least one package directory exists under `packages/`.
4. Each function either:
   - Is a single file with a recognized extension, or
   - Is a directory containing at least one source file.

If any of these are false, stop and ask the user to fix or clarify. Do not modify their project without permission.

### 4.6 Runtime kinds

Use only the runtime strings listed below. The authoritative source is `doctl serverless status --languages` against a connected namespace — consult it when unsure or when a new version is suspected. Do not invent runtime strings.

**Supported runtimes:**

- **Go:** `go:1.17`, `go:1.20`, `go:1.24`, `go:1.25`
- **Node.js:** `nodejs:14`, `nodejs:18`, `nodejs:22`, `nodejs:24`
- **PHP:** `php:8.0`, `php:8.2`, `php:8.3`, `php:8.4`, `php:8.5`
- **Python:** `python:3.9`, `python:3.11`, `python:3.12`, `python:3.13`

**Recommended defaults** when the user hasn't specified a version: use the highest stable minor of each language family — `go:1.25`, `nodejs:24`, `php:8.5`, `python:3.13`.

If the user requests a runtime not on this list, stop and tell them which runtimes are actually supported. Do not silently substitute.

To re-verify this list at any time:

```bash
doctl serverless status --languages
```

Requires `doctl` to already be connected (Phase 2 complete).

### 4.7 The `web` and `webSecure` fields

- Set `web: true` if the function should be HTTP-accessible as a web endpoint.
- Set `web: true` + `webSecure: true` for basic-auth-protected web endpoints.
- Omit both for functions invoked only via the API or triggers.

Ask the user if unclear. Default to `web: true` for anything that looks like an HTTP handler; default to no web access for anything that looks like scheduled / trigger-driven work.

---

## 5. Phase 4 — Deploy

### 5.1 Decide: local build or remote build

**What `--remote-build` actually does.** It specifies *where* the build step runs (DigitalOcean-managed runtime container vs. the user's machine). It does **not** by itself cause a build step to exist. A build step only runs when the deployer finds a recognized build trigger in a function's directory — see the dominance rule in Section 4.3. Without a trigger, the deployer produces a passthrough zip regardless of whether `--remote-build` is passed. This is the single most common reason a deploy appears to succeed but the function is broken at cold-start.

Before choosing local vs. remote, confirm that each function with third-party deps has a valid build trigger for its runtime (Python / PHP need `build.sh`; Node.js needs `package.json` or `build.sh`; Go needs `go.mod`). If any function is missing the right trigger, **fix Section 4.3 first** — no flag choice will save you.

Once the triggers are in place, decide between local and remote build using this order:

1. **Use remote build** (`--remote-build`) when any of these is clearly true:
   - Any function's directory contains a dependency file: `package.json` with non-empty `dependencies`, `requirements.txt`, `go.mod`, or `composer.json`
   - Any function's directory contains a `build.sh` or `build.cmd` script
   - User explicitly asked for remote build
   - Runtime is Go and there is no user-provided build script (the deployer already defaults Go to remote in this case; passing the flag is harmless and makes intent explicit)

2. **Use local build** (no flag) when all functions are single-file with no dependencies and no build script.

3. **If the situation is ambiguous** — for example, a project has a `build.sh` you cannot interpret, an unusual layout, or you are unsure whether the user's machine has the right toolchain — **stop and ask the user**:
   > "Your project has <describe what you found>. Should I deploy using remote build (runs the build on DigitalOcean's infrastructure — slower but more reliable, and doesn't require local tooling) or local build (faster, but requires <list the toolchains needed> to be installed locally)?"

   Wait for the user's answer. Do not guess.

Local build is faster (no upload, no cold-start of a builder action) but depends entirely on the user's machine having the correct toolchains, versions, and native libraries. Remote build is slower but portable.

### 5.2 Run deploy

```bash
doctl serverless deploy <project-path> -o json [--remote-build]
```

Note: `-o json` (alias for `--output json`) is a **global `doctl` flag**, not a deploy-specific one. It must come before or after the `deploy` subcommand, not between `deploy` and `<project-path>` in a position that `doctl` rejects. The form shown above works.

Other useful flags on `deploy` (surface to the user only when relevant):

- `--incremental` — deploy only changes since last deploy (uses `.deployed/versions.json` in the project)
- `--include <pattern>` / `--exclude <pattern>` — limit to or skip specific packages/functions
- `--env <path>` — path to a runtime environment file
- `--build-env <path>` — path to a build-time environment file
- `--verbose-build` — show detailed build output (useful for debugging remote builds)
- `--yarn` — use yarn instead of npm for Node.js builds

Capture stdout and stderr separately. The JSON result is on stdout; the human-readable build transcript is on stderr.

### 5.3 Parse the result

The JSON on stdout is a `DeployResponse` object. Key fields:

- `successes`: array of deployed entities with their names and versions
- `failures`: array of per-action errors with messages
- `namespace`: the namespace that was deployed to

Branch on the result:

- **If `failures` is empty and `successes` is non-empty:** the deploy succeeded. Go to Phase 5.
- **If `failures` is non-empty:** report each failure to the user with the exact error message. Do not try to auto-fix unless the error is obviously a typo in `project.yml` that you just wrote (in which case: fix it, note what you changed, re-run deploy once, and stop if it fails again).
- **If the command exited non-zero with no parseable JSON:** report the stderr transcript verbatim. Do not hallucinate a reason.

### 5.4 Handle remote-build duration

Remote builds can take 30 seconds to several minutes. Do not assume the command is hung. If progress messages appear on stderr, surface them to the user periodically so they know work is happening.

**Do not wrap the deploy command with `time`, a timer, or any measurement harness.** Just run it and report progress from stderr. Duration is not a signal we use.

---

## 6. Phase 5 — Verification

`doctl serverless deploy` reporting success means the artifact reached the namespace — it does **not** guarantee the function initializes or runs. A function whose `build.sh` is missing, broken, or skipped will deploy "successfully" but fail at cold-start. Invocation is the only way to confirm the function actually works.

However, **invocation executes the deployed function with real credentials against real infrastructure.** If the function has side effects — sending SMS, emails, calls to paid APIs, writes to databases, money-moving operations — invocation will trigger those side effects with whatever payload you supply. For this reason, the agent must **not** invoke automatically. Always propose the verification invocation and get explicit user consent first.

### 6.1 Confirm the function exists

Call `functions-get-action` with the namespace and action name. Confirm the returned action has:

- The expected runtime
- The expected `web` annotation (if applicable)
- A recent `updated` timestamp

### 6.2 Propose the verification invocation (requires user consent)

Do **not** call `functions-invoke-action` automatically. Instead, prepare a proposal and ask the user:

1. Pick a minimal, sensible test payload:
   - For functions with no required parameters, use `{}`.
   - For functions with required parameters, use placeholder values that will not cause an expensive or irreversible side effect (for example, for an SMS-sending function propose a payload targeting the user's own number, not a random one).
2. Present the proposal to the user in roughly this form:
   > "Deploy completed. I'd like to verify the function actually initializes and runs by invoking it once with this payload: `<payload>`. Note: this will execute the function for real, so any side effects (<briefly list what you think the function does — SMS, email, DB write, external API call, etc. — or say "I don't see obvious side effects" if the code looks pure>) will happen. Should I invoke, skip verification, or invoke with a different payload?"
3. Wait for the user's answer. Three branches:
   - **Invoke as proposed** — proceed to 6.3.
   - **Invoke with a different payload** — use the payload the user specifies, then proceed to 6.3.
   - **Skip verification** — go directly to 6.4 and report the deploy as completed-but-unverified.

Remember the user's preference for the rest of the session (see Section 7). If the user says "just verify automatically from now on" or "don't verify unless I ask," carry that policy forward for subsequent deploys in the same conversation.

### 6.3 Inspect the activation and classify the result

**Only reach this step if the user consented to invocation in 6.2.**

Call `functions-list-activations` with `FunctionName` set to the just-deployed function. Get the most recent activation and call `functions-get-activation-logs` and `functions-get-activation-result` on its ID.

Classify the outcome:

- **PASS** — the activation's `statusCode` is 0 (success) and the result matches what the function should return. Continue to 6.4.
- **FAIL — initialization error** — any of these symptoms:
  - Invoke returns HTTP 502 or a body containing "The function did not initialize properly"
  - Activation logs contain `ModuleNotFoundError` (Python), `Error: Cannot find module` (Node.js), `Fatal error: Uncaught Error: Class ... not found` (PHP), or equivalent missing-dependency errors
  - Activation `statusCode` is 1 with a dependency-loading error

  **Root cause is almost always a missing or broken build step.** Go back to Section 4.3:
  - Python / PHP: verify a `build.sh` exists in the function's directory and is executable (`chmod +x`). If it's missing, create it using the template in Appendix A and redeploy with `--remote-build`.
  - Node.js: verify `package.json` lists the missing module under `dependencies` (not `devDependencies`). Redeploy with `--remote-build`.
  - Go: verify `go.mod` declares the module and all imports resolve with `go mod tidy` locally before redeploying.

  Report the diagnosis to the user before making any changes. Make one remediation attempt, then stop if it still fails (per Section 7: never rerun a failed deploy more than once without concrete changes or user permission).

- **FAIL — runtime error** — the function initialized but threw an exception during execution. This is a code bug, not a deploy bug. Report the exact stack trace from the logs to the user and stop; do not attempt to fix user code without permission.

### 6.4 Report summary

Reach this step in one of two cases:

- **Verified:** 6.3 classified the outcome as PASS.
- **Unverified:** the user declined the verification invocation in 6.2.

End with a concise summary:

- What was deployed (function names, package)
- Namespace and region
- Web URL if applicable
- If verified: the test payload you invoked with, the result, and the activation ID
- If unverified: an explicit note that the function was deployed but not invoked, and a short suggestion of how the user can verify themselves (e.g. "You can run `doctl serverless functions invoke <name> -p key:value` or call the `functions-invoke-action` MCP tool when you're ready")
- Any warnings from the deploy (e.g. skipped actions in incremental mode)

---

## 7. Error handling rules

- **Never hide errors.** If a step fails, report the exact error.
- **Never fabricate commands.** If you don't know the exact flag, stop and ask the user (or ask to run `--help` first).
- **Never rerun a failed deploy more than once** without either (a) making a concrete change or (b) getting user permission.
- **Never delete user files** without explicit permission. This includes `project.yml`, `node_modules`, `.deployed/`, or anything else in the project directory.
- **Never leak secrets.** Access key secrets, API tokens, and credentials must not appear in messages to the user, even in error reports.
- **Remember user decisions within the session.** Once the user has answered a question in this conversation — which namespace to use, local vs. remote build, `web` vs. non-web, runtime choice, etc. — do not ask the same question again for subsequent deploys in the same session. Reuse the previous answer. Only re-ask if the user changes the project in a way that invalidates the prior decision (e.g. they add dependencies to a previously dep-free project).

---

## 8. Stop conditions (ask the user; do not guess)

Stop and ask the user when:

- `doctl` is not installed and installation requires their consent
- `doctl auth init` has not been run
- Multiple namespaces exist and none is obviously the right one
- Zero namespaces exist (need a region)
- The user's project is ambiguous (e.g. no clear entrypoint)
- The required runtime is not on the supported list
- A deploy failure is not obviously a typo you can fix
- The user's intent is unclear about `web` / `webSecure` for a function
- The choice between local build and remote build is ambiguous (see section 5.1)
- Any step requires `sudo`
- Before invoking a deployed function for verification (Section 6.2) — propose the payload and wait for explicit consent. Never invoke automatically, because invocation triggers whatever side effects the function has (SMS, email, paid API calls, DB writes).

---

## 9. Cleanup (post-deploy)

- If you created an access key in step 3.3 (Option B), leave it alone — it has a 24h expiry and will self-clean. Do **not** delete it immediately after deploy in case the user re-runs.
- If you scaffolded a new project, leave all files on disk. Do not delete `.deployed/versions.json` — the deployer uses it for incremental deploys.
- If the deploy failed and you created a new project directory, **ask before deleting** — the user may want to inspect and fix.

---

## Appendix A — Minimal `project.yml` templates

### Node.js, single file, web endpoint

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: hello
        runtime: 'nodejs:20'
        web: true
```

### Python, function with dependencies

`project.yml`:

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: fetch
        runtime: 'python:3.11'
        web: true
```

Directory layout:

```
<project-path>/
  project.yml
  packages/
    default/
      fetch/
        __main__.py
        requirements.txt
        build.sh          # REQUIRED — without this, pip install never runs
```

`packages/default/fetch/build.sh` (make executable with `chmod +x`):

```bash
#!/bin/bash
set -e
virtualenv virtualenv
source virtualenv/bin/activate
pip install -r requirements.txt
deactivate
```

This template matches the deployer's e2e fixture at `functions-deployer/e2e/remote-build-python/packages/test-remote-build-python/default/build.sh`.

### PHP, function with dependencies

`project.yml`:

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: handler
        runtime: 'php:8.4'
        web: true
```

Directory layout:

```
<project-path>/
  project.yml
  packages/
    default/
      handler/
        index.php
        composer.json
        build.sh          # REQUIRED — without this, composer install never runs
```

`packages/default/handler/build.sh` (make executable with `chmod +x`):

```bash
#!/bin/bash
set -e
composer install
```

This template matches the deployer's e2e fixture at `functions-deployer/e2e/remote-build-php/packages/test-remote-build-php/default/build.sh`.

### Node.js, function with dependencies

`project.yml`:

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: greet
        runtime: 'nodejs:22'
        web: true
```

Directory layout:

```
<project-path>/
  project.yml
  packages/
    default/
      greet/
        index.js
        package.json      # deps under "dependencies"
```

No `build.sh` required — the deployer's `npmBuilder` runs `npm install` automatically when it sees `package.json`. Add a `build.sh` only if you need a custom build step (TypeScript, bundling, etc.), and if you do, the script **replaces** the automatic `npm install` — your script must run `npm install` itself.

### Go function

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: handler
        runtime: 'go:1.22'
        web: true
```

With `packages/default/handler/main.go` and `packages/default/handler/go.mod`. Go functions with no local `build.sh` are compiled remotely by default; you do not need to pass `--remote-build` explicitly for that case, though doing so is harmless.

### Scheduled trigger (no web)

```yaml
parameters: {}
environment: {}
packages:
  - name: default
    actions:
      - name: cron-task
        runtime: 'nodejs:20'
        web: false
```

Triggers themselves are created separately via the `functions-create-trigger` MCP tool after deploy; they are not declared in `project.yml` in this spec.

---

## Appendix B — Supported runtime reference

The authoritative source at any given moment is `doctl serverless status --languages` against a connected namespace. The table below is the current verified snapshot; re-verify before shipping if a significant amount of time has passed.

| Language | Runtime kinds | File extensions | Dependency file |
|---|---|---|---|
| Go | `go:1.17`, `go:1.20`, `go:1.24`, `go:1.25` | `.go` | `go.mod` |
| Node.js | `nodejs:14`, `nodejs:18`, `nodejs:22`, `nodejs:24` | `.js`, `.mjs` | `package.json` |
| PHP | `php:8.0`, `php:8.2`, `php:8.3`, `php:8.4`, `php:8.5` | `.php` | `composer.json` |
| Python | `python:3.9`, `python:3.11`, `python:3.12`, `python:3.13` | `.py` | `requirements.txt` |

**Defaults when the user doesn't specify a version:** `go:1.25`, `nodejs:24`, `php:8.5`, `python:3.13`.

---

## Appendix C — Commands reference

Verified against `doctl` 1.154.0. See Appendix D for the full verification log.

```bash
# Preflight
doctl version
doctl auth list
doctl serverless status

# Install the serverless plugin (required once per doctl install)
doctl serverless install

# Connect (preferred — uses existing doctl API-token auth; requires function:admin scope)
doctl serverless connect                  # auto-connect if only one namespace
doctl serverless connect <hint>           # partial match against label or UUID

# Connect (fallback — via access key; value format is dof_v1_<id>:<secret>)
doctl serverless connect <hint> --access-key dof_v1_<id>:<secret>

# Scaffold a starter project
doctl serverless init <project-path>
doctl serverless init <project-path> --language <nodejs|python|go|php>

# List supported runtimes (requires an active connection)
doctl serverless status --languages

# Deploy (global -o json for structured output; flags may be in any order)
doctl serverless deploy <project-path> -o json
doctl serverless deploy <project-path> -o json --remote-build
doctl serverless deploy <project-path> -o json --incremental
doctl serverless deploy <project-path> -o json --remote-build --verbose-build

# Inspect
doctl serverless status
doctl serverless functions list
doctl serverless activations list
```

---

## Appendix D — Verification log

All items verified against `doctl` 1.154.0:

1. Access-key connect flag → `--access-key`
2. Deploy output format → global `-o` / `--output` flag with value `json` (`-o json`)
3. Connect arg form → positional `<hint>`, partial match against label or UUID; optional if only one namespace
4. Remote-build flag → `--remote-build`
5. Incremental flag → `--incremental` exists on `deploy`
6. Access-key value format → `dof_v1_<access-key-id>:<secret>` as a single joined string
7. Supported runtime kinds → Go 1.17/1.20/1.24/1.25, Node.js 14/18/22/24, PHP 8.0/8.2/8.3/8.4/8.5, Python 3.9/3.11/3.12/3.13

8. **Build-trigger dominance rule** → `build.sh` (or `build.cmd`) > `package.json` > `.include` > `identify` (passthrough zip). Verified in `functions-deployer/src/finder-builder.ts:1011-1063` (`findSpecialFile`) and confirmed by the e2e fixtures `functions-deployer/e2e/remote-build-python/...` and `functions-deployer/e2e/remote-build-php/...`. Key consequence: Python and PHP have **no automatic dependency builder** — without a `build.sh`, `requirements.txt` / `composer.json` are zipped as plain files and no install runs, producing a broken deploy at cold-start. Node.js has a special `npmBuilder` that fires on `package.json`. Go has its own remote-build path triggered by `go.mod`. This rule is the root cause behind the most common "deploy succeeded but function returns 502 / ModuleNotFoundError" failure.

Convention choice (not a verification):

9. Agent-created access keys use the name prefix `mcp-agent-`. The prefix `mcp-do-` is **reserved** and managed by this MCP server's own `OWResolver` (see `pkg/registry/functions/ow_resolver.go`) for its internal OpenWhisk client — agents must never create or delete keys with that prefix. The two prefixes together form a clear namespace split: server owns `mcp-do-*`, agents own `mcp-agent-*`, and everything else is user/third-party territory.
