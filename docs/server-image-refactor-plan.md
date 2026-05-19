# Server Image Resolution Refactor Plan

**Goal:** Establish `~/.kubecraft/config` as the single runtime source of truth for server image, while keeping `--server-image` as an explicit one-shot override. Remove the `KUBECRAFT_SERVER_IMAGE` env var override to eliminate hidden behavior.

---

## Problem Statement

Server image can currently be set in too many places, creating ambiguity:

- Build-time default (`config.ServerImage`)
- Environment variable (`KUBECRAFT_SERVER_IMAGE`)
- CLI flag (`--server-image`)

This makes debugging difficult and breaks the principle of a single source of truth.

---

## Target Architecture (Option 2)

Resolution precedence (deterministic):

1. **CLI flag** `--server-image` — one-shot override for this invocation only
2. **Config file** `~/.kubecraft/config.serverImage` — persistent runtime source of truth
3. **Fallback constant** `config.ServerImage` — only if config value is empty (legacy configs)

Removed:
- `KUBECRAFT_SERVER_IMAGE` env var override (hidden, non-obvious)

---

## File-by-File Changes

### `internal/config/config.go`

- Add field to `Config` struct:
  ```go
  ServerImage string `yaml:"serverImage"`
  ```
- Add helper method:
  ```go
  func (c *Config) EffectiveServerImage() string
  ```
  - Returns `c.ServerImage` if non-empty
  - Else returns package-level `config.ServerImage` fallback constant

### `internal/cli/init.go`

- During `buildInitConfig`, if `cfg.ServerImage` is empty, set it to the fallback constant:
  ```go
  if cfg.ServerImage == "" {
      cfg.ServerImage = config.ServerImage
  }
  ```
- This ensures the config file always contains an explicit `serverImage` value after init.

### `internal/cli/server/create.go`

- Remove `KUBECRAFT_SERVER_IMAGE` env var lookup entirely
  - Delete `serverImageEnvVar` constant
  - Delete `os.Getenv(serverImageEnvVar)` fallback block
- Replace image resolution logic:
  ```go
  resolvedImage := cli.AppConfig.EffectiveServerImage()
  if serverImage != "" {
      resolvedImage = serverImage // flag override
  }
  ```
- Before creating server, print:
  ```
  Using server image: <resolved-image>
  ```
- Pass `resolvedImage` into `CreateServer(...)`

### `internal/k8s/server.go`

- Keep existing empty-string guard as defensive fallback:
  ```go
  if serverImage == "" {
      serverImage = config.ServerImage
  }
  ```
- No behavior change required at this layer.

### Documentation Updates

- `AGENTS.md`
  - Remove `KUBECRAFT_SERVER_IMAGE` env var mention
  - Update to: defaults come from config file; override via `--server-image`
- `README.md`
  - Remove env var examples
  - Keep `--server-image` flag examples
- `CLAUDE.md`
  - Align with new resolution policy

---

## Tests to Add / Update

### `internal/config/config_test.go`

Add `TestConfigEffectiveServerImage`:
- subtest: returns config value when set
- subtest: returns fallback constant when empty

### `internal/cli/server/create_test.go`

- Remove any env var-based test setup
- Add precedence tests:
  1. flag set → uses flag value
  2. no flag + config value set → uses config value
  3. no flag + config empty → uses fallback constant
- Optionally assert that `Using server image: ...` is printed

### `internal/cli/init_test.go`

- Add test: init sets `serverImage` on fresh config
- Add test: re-init does not overwrite existing custom `serverImage`

---

## Backward Compatibility

- Existing configs without `serverImage` continue to work via fallback constant
- First re-init (or server create with empty config value) implicitly populates `serverImage` in config
- No CLI command signatures change
- No breaking change for users who previously used env var (they can switch to config or flag)

---

## Validation Checklist

- [ ] `make test` passes
- [ ] `go test -p 1 -tags=integration ./internal/...` passes (with cluster)
- [ ] Manual smoke test:
  - `kubecraft init --ip <ip>`
  - `kubecraft register --username <name>`
  - `kubecraft server create s1` → uses config image, prints resolved image
  - `kubecraft server create s2 --server-image <custom>` → uses override, prints resolved image
- [ ] Confirm env var no longer influences behavior:
  - `export KUBECRAFT_SERVER_IMAGE=some-image`
  - run create without flag → should still use config value, not env var

---

## Acceptance Criteria

- No runtime image resolution from environment variable
- `~/.kubecraft/config` is the primary persistent source of truth
- `--server-image` remains a clear, explicit one-shot override
- Behavior is transparent in CLI output and docs
