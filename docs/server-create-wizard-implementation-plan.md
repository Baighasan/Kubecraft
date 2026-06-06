# Server Create Wizard Implementation Plan

## Objective

Implement an interactive wizard for `kubecraft server create` that gives users guided control over server setup through exactly four questions:

1. Minecraft version (strict menu)
2. Server name (validated input)
3. Game mode (strict menu)
4. Max players (strict menu, `1-50`)

The wizard must auto-start when users run `kubecraft server create` with no positional arguments.

## Current Status

- Phase 1 is implemented in code.
- Phase 2 is implemented in code, including the intentional runtime wiring deviation for `server.properties` (`gamemode`, `max-players`).
- Phase 3 is implemented in code, including approved deviations:
  - image resolution helper to remove mutable global runtime state.
  - zero-arg wizard dispatch wiring (`runCreateWizard()`).
- Phase 4 is implemented in code with approved deviations:
  - interactive 4-question wizard prompts.
  - reader/writer-backed prompt helpers while runtime remains wired to `os.Stdin`/`os.Stderr`.
  - strict bounded numeric max-players input (`1-50`) instead of rendering 50 menu rows.
- Phase 5 confirmation behavior has been intentionally pulled into Phase 4 and implemented:
  - summary + `Proceed? (y/N)` prompt before mutations.
  - cancellation exits successfully (`nil`) without creating resources.
- Validation is implemented in CLI (`validateCreateInput`) for server name, version, game mode, and max players.
- Local validation is complete (`go test ./internal/cli/server`, `make test`, and `make build`).
- Phase 6 create-pipeline wiring is implemented in code (CLI input -> `k8s.ServerSpec` -> `CreateServer(...)`).
- Phase 7 test coverage is implemented in code, including:
  - wizard composition cancellation/confirmation behavior tests in CLI.
  - max-player boundary validation coverage in CLI.
  - custom `ServerSpec` env assertion test in k8s integration tests.
- Phase 8 documentation alignment is implemented in `README.md` (wizard launch, built-in version list, cancellation semantics).
- Remaining work is integration validation in a cluster-enabled environment (`go test -v -race -p 1 -tags=integration ./internal/...`).

## Locked Product Decisions

- `kubecraft server create` (no args) starts wizard automatically.
- `kubecraft server create <server-name>` remains supported and non-interactive.
- Version input is not free text; user selects from a curated version list.
- Game mode is strict menu: `survival`, `creative`, `adventure`, `spectator`.
- Max players is constrained to `1-50`.
- Final confirmation cancel exits successfully (status code `0`).
- No new non-wizard flags for version/game mode/max players in this phase.
- Existing `--server-image` behavior remains unchanged.

## Scope

### In Scope

- CLI wizard flow for `server create`.
- Data model changes to pass selected options into Kubernetes object creation.
- Validation and sane defaults for non-interactive fallback path.
- Tests and docs updates for new behavior.

### Out of Scope

- Dynamic fetching of Minecraft versions from remote APIs.
- New command flags for `--version`, `--game-mode`, `--max-players`.
- Resource tuning (CPU/RAM/storage) questions in wizard.
- Broader UX redesign across other commands.

## Phase 0 - Baseline and Safety Checks

### Goals

- Capture current behavior and avoid regressions while refactoring.

### Tasks

1. Confirm current `server create` flow and where defaults are hardcoded.
2. Confirm current test baselines:
   - `make test`
   - integration test invocation pattern remains unchanged (`-p 1`).
3. Identify all `CreateServer(...)` callsites that need signature updates.

### Exit Criteria

- Known list of files and tests impacted by signature/refactor.

## Phase 1 - Introduce Server Spec Model

### Goals

- Decouple server runtime options from hardcoded environment values.

### Tasks

1. Add `ServerSpec` in `internal/k8s` (or neighboring package-local type) with:
   - `Version string`
   - `GameMode string`
   - `MaxPlayers int`
2. Update `CreateServer(...)` signature to accept `ServerSpec`.
3. Replace hardcoded env vars in StatefulSet container env:
   - `VERSION`
   - `GAME_MODE`
   - `MAX_PLAYERS`
4. Add `DefaultMinecraftVersion`, `DefaultGameMode`, and `DefaultMaxPlayers` to `internal/config/constants.go` in this phase (intentional sequencing deviation from Phase 2) to avoid temporary hardcoded defaults and match existing config patterns.
5. Add a `DefaultServerSpec()` helper in `internal/k8s` that builds from config defaults.
6. Keep `EULA`, `JAVA_MEMORY`, and resource limits unchanged.

### Intentional Deviation Note

- Although defaults were originally listed for Phase 2, introducing only the three default constants in Phase 1 keeps runtime defaults centralized and avoids duplicate temporary literals in the k8s layer.
- Allowed lists (`AllowedGameModes`, `AllowedMinecraftVersions`) and max-player bounds remain in Phase 2 as planned.

### Risks

- Compile breaks across all existing callsites/tests.

### Exit Criteria

- Code compiles with new signature and default spec values wired.

## Phase 2 - Define Defaults and Allowed Option Sets

### Goals

- Centralize wizard and non-wizard defaults in config constants.
- Ensure selected runtime values are actually applied by the Minecraft process.

### Tasks

1. Add constants to `internal/config/constants.go`:
    - `MinMaxPlayers`, `MaxMaxPlayers`
2. Add allowed values:
    - `AllowedGameModes = []string{"survival", "creative", "adventure", "spectator"}`
    - `AllowedMinecraftVersions = []string{...}` (curated static list)
3. Ensure defaults are members of allowed lists.
4. Runtime wiring deviation (intentional): update `docker/minecraft/start.sh` to generate a minimal `server.properties` that includes:
   - `gamemode=${GAME_MODE}`
   - `max-players=${MAX_PLAYERS}`
   - keep this minimal and avoid uncommenting properties backed by currently undefined env vars.

### Intentional Deviation Note

- Phase 2 originally focused on config-only constants.
- Adding minimal `server.properties` generation in this phase closes the gap between pod env vars and actual Minecraft runtime behavior for game mode and max players.

### Version List Strategy

- Start with a short curated stable list for UX clarity.
- Keep list static in this phase; future phase may fetch dynamically.

### Exit Criteria

- Single source of truth exists for defaults and menus.
- Runtime config file generation applies `GAME_MODE` and `MAX_PLAYERS` values.

## Phase 3 - CLI Flow Refactor for Dual Mode

### Goals

- Support both wizard mode (zero args) and legacy direct mode (one arg).

### Tasks

1. Update `create` command arg policy from `ExactArgs(1)` to `MaximumNArgs(1)`.
2. Route execution:
   - `len(args) == 0` -> wizard path
   - `len(args) == 1` -> non-interactive path with defaults
3. Introduce internal input struct in CLI layer, e.g. `createInput`:
   - `ServerName`
   - `Version`
   - `GameMode`
   - `MaxPlayers`
4. Add resolver helpers:
    - `buildDefaultCreateInput(serverName string)`
    - `executeCreateWithInput(input createInput)`

### Intentional Deviations (Approved)

- Add `resolveServerImage(flagValue string)` in CLI layer and stop mutating package-global `serverImage` during execution.
- Rationale: avoids cross-invocation/test state bleed while preserving existing precedence (`--server-image` -> `KUBECRAFT_SERVER_IMAGE` -> build-time default).
- Add a temporary `runCreateWizard()` placeholder in Phase 3 for zero-arg dispatch wiring; full prompt behavior remains Phase 4.

### Exit Criteria

- Command supports both entry modes without behavior ambiguity.

## Phase 4 - Implement Wizard Prompts + Confirmation

### Goals

- Deliver the interactive guided setup with strict selection behavior.
- Ensure no cluster mutation occurs before explicit confirmation.

### Tasks

1. Implement wizard function, e.g. `runCreateWizard()`.
2. Prompt sequence:
    - Q1 version: numeric/menu index select from `AllowedMinecraftVersions`
    - Q2 server name: stdin text input
    - Q3 game mode: menu select from `AllowedGameModes`
    - Q4 max players: menu select from `1..50`
3. Normalize selected values into `createInput`.
4. Keep prompts and status output on stderr for CLI consistency.
5. Add confirmation block immediately after question flow (pulled forward from Phase 5):
   - print summary of four selected fields
   - prompt `Proceed? (y/N)`
   - on non-confirmation, print cancellation message and return `nil` (success exit)

### Approved Deviations

- Use reader/writer-backed internal prompt helpers for testability while wiring runtime calls to `os.Stdin` and `os.Stderr`.
- For max players, use strict bounded numeric selection (`1-50`) instead of rendering 50 menu rows.
- Pull confirmation forward from Phase 5 into this phase to satisfy "no hidden side effects before confirmation".

### UX Notes

- Keep prompts explicit and forgiving (re-prompt on invalid menu selection).
- Avoid hidden side effects before final confirmation.

### Exit Criteria

- Wizard can collect all four values reliably in terminal.
- Wizard cancellation at confirmation exits successfully and does not mutate cluster resources.

## Phase 5 - Validation and Confirmation

### Goals

- Prevent invalid creates and guarantee safe cancellation behavior.

### Tasks

1. Add `validateCreateInput(...)`:
   - Server name via existing `ValidateServerName(...)`
   - Version must exist in `AllowedMinecraftVersions`
   - Game mode must exist in `AllowedGameModes`
   - Max players in `1..50`
2. Keep confirmation behavior implemented in Phase 4 and validate it with tests.

### Exit Criteria

- Invalid data rejected early; cancellation is non-destructive and success-exit.

## Phase 6 - Wire Create Pipeline With Spec

### Goals

- Ensure wizard/default inputs are used by actual provisioning.

### Tasks

1. Convert CLI input -> `k8s.ServerSpec` before calling `CreateServer(...)`.
2. Preserve existing pipeline order:
   - validate name / existence
   - capacity check
   - nodeport allocate
   - create service + statefulset
   - readiness wait
3. Keep error messaging style aligned with existing command output.

### Exit Criteria

- Created StatefulSet env reflects user-selected values.

## Phase 7 - Tests

### Goals

- Lock correctness and prevent regressions.

### Tasks

1. CLI tests (`internal/cli/server/create_test.go`):
   - zero-arg dispatch to wizard
   - one-arg direct path uses defaults
   - validation failures for name/version/mode/max players
   - confirmation cancel path returns success and does not create server
2. K8s integration tests (`internal/k8s/server_test.go`):
   - update all `CreateServer(...)` callsites with default `ServerSpec`
   - add assertion test for custom spec env values in StatefulSet
3. Run unit scope via `make test`.
4. If cluster available, run integration tests with serial flag.

### Exit Criteria

- Unit tests pass; integration tests compile and pass in cluster-enabled environments.

## Phase 8 - Documentation and Operator Notes

### Goals

- Keep user docs aligned with new behavior.

### Tasks

1. Update `README.md` command section:
   - show `kubecraft server create` launches wizard
   - preserve direct mode example with explicit name
2. Add short note that version is selected from built-in list.
3. Document cancellation semantics (`Proceed?` cancel exits successfully).

### Exit Criteria

- README reflects actual UX and command behavior.

## Phase 9 - Rollout and Future Follow-Ups

### Immediate Follow-Ups (post-merge)

1. Add optional non-wizard flags (`--version`, `--game-mode`, `--max-players`) for automation parity.
2. Consider reusable prompt helpers for consistency across commands.
3. Consider dynamic version discovery (remote metadata + local fallback).

### Long-Term Follow-Ups

1. Add wizard support for advanced options (difficulty, PvP, seed, whitelist).
2. Add profile presets (casual, creative-build, event-server).
3. Consider machine-readable output option for script pipelines.

## Acceptance Criteria Checklist

- `kubecraft server create` starts wizard and asks exactly four setup questions.
- `kubecraft server create <name>` remains functional and non-interactive.
- Wizard uses strict menu selections for version, game mode, and max players.
- Max players enforcement is exactly `1-50`.
- Cancel at confirmation exits with success and does not mutate cluster resources.
- Server pod env vars (`VERSION`, `GAME_MODE`, `MAX_PLAYERS`) reflect selected/default values.
- Existing image override behavior remains intact.
- Tests and README are updated accordingly.
