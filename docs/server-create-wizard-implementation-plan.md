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
- Defaults (`DefaultMinecraftVersion`, `DefaultGameMode`, `DefaultMaxPlayers`) were intentionally moved into Phase 1 and are complete.
- Remaining phases are pending.

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

### Tasks

1. Add constants to `internal/config/constants.go`:
   - `MinMaxPlayers`, `MaxMaxPlayers`
2. Add allowed values:
   - `AllowedGameModes = []string{"survival", "creative", "adventure", "spectator"}`
   - `AllowedMinecraftVersions = []string{...}` (curated static list)
3. Ensure defaults are members of allowed lists.

### Version List Strategy

- Start with a short curated stable list for UX clarity.
- Keep list static in this phase; future phase may fetch dynamically.

### Exit Criteria

- Single source of truth exists for defaults and menus.

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

### Exit Criteria

- Command supports both entry modes without behavior ambiguity.

## Phase 4 - Implement Wizard Prompts (4 Questions)

### Goals

- Deliver the interactive guided setup with strict selection behavior.

### Tasks

1. Implement wizard function, e.g. `runCreateWizard()`.
2. Prompt sequence:
   - Q1 version: numeric/menu index select from `AllowedMinecraftVersions`
   - Q2 server name: stdin text input
   - Q3 game mode: menu select from `AllowedGameModes`
   - Q4 max players: menu select from `1..50`
3. Normalize selected values into `createInput`.
4. Keep prompts and status output on stderr for CLI consistency.

### UX Notes

- Keep prompts explicit and forgiving (re-prompt on invalid menu selection).
- Avoid hidden side effects before final confirmation.

### Exit Criteria

- Wizard can collect all four values reliably in terminal.

## Phase 5 - Validation and Confirmation

### Goals

- Prevent invalid creates and guarantee safe cancellation behavior.

### Tasks

1. Add `validateCreateInput(...)`:
   - Server name via existing `ValidateServerName(...)`
   - Version must exist in `AllowedMinecraftVersions`
   - Game mode must exist in `AllowedGameModes`
   - Max players in `1..50`
2. Add confirmation block before cluster mutations:
   - Print summary of 4 selected fields
   - Prompt `Proceed? (y/N)`
3. Cancel behavior:
   - If not confirmed, print cancellation message
   - return `nil` (exit success)

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
