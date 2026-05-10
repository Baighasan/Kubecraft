# Distribution Migration Plan: From Pre-configured Binary to BYO Cluster

**Status:** Phase 1 complete — Phase 2 (Init Command) pending  
**Goal:** Ship a single generic CLI artifact with no environment-specific endpoint/IP baked in, so users bring their own Kubernetes cluster.

---

## Scope Contract (Locked)

- User provides only one input at setup: public cluster IP.
- Fixed ports: Kubernetes API `6443`, registration `30099`.
- Network model: public NodePort access for Minecraft (`30000-30015`).
- Registration transport stays HTTP for now; HTTPS migration planned later.
- TLS behavior: try secure first, fallback to insecure, persist `tlsInsecure=true`.
- Onboarding remains explicit: `kubecraft init` then `kubecraft register`.
- v1 cluster compatibility: one public IP serves both API and NodePort traffic.

---

## Old UX vs New UX

### Old Flow (Today)

```
1. Operator builds env-specific binary:
   make build-prod PROD_ENDPOINT=<host:6443> PROD_NODE_ADDRESS=<public-ip>

2. User downloads and installs that special binary

3. User runs:
   kubecraft register --username <name>
   → POST to registration service at baked-in endpoint:30099

4. User runs server commands:
   kubecraft server create <name>
   → API calls to baked-in endpoint:6443
   → "Server ready at <baked-in-node-address>:<port>"

Problems:
- Binary is tied to one cluster
- Endpoint change = rebuild + redistribute
- Release pipeline embeds infra secrets in artifacts
- No generic artifact for public distribution
```

### New Flow (Target)

```
1. User installs one generic CLI artifact (from GitHub releases)

2. User runs:
   kubecraft init --ip <public-ip>
   → Derives API endpoint: https://<ip>:6443
   → Derives registration: http://<ip>:30099/register
   → Derives node address: <ip>
   → TLS probe: strict first, fallback to insecure if needed
   → Persists cluster config to ~/.kubecraft/config

3. User runs:
   kubecraft register --username <name>
   → Uses runtime-configured registration endpoint
   → Returns token, saved to config

4. User runs server commands:
   kubecraft server create <name>
   → API calls to runtime-configured endpoint
   → "Server ready at <runtime-ip>:<port>"

Benefits:
- One artifact for all clusters
- No infra data in release binaries
- Clear setup contract
- Self-documenting config
```

### User-Input Reduction

| Before | After |
|--------|-------|
| Endpoint + node address supplied indirectly at build/release time | Single runtime input (`--ip`) once per machine/profile |
| Build-time ldflags: `PROD_ENDPOINT`, `PROD_NODE_ADDRESS` | Runtime config: derived from `--ip` |
| Release pipeline needs env vars | Release pipeline is env-agnostic |

---

## Detailed Phase Plan

### Phase 0: UX/Contract Freeze

**Goal:** Define final command contract and error messaging before writing code.

- [x] Define final CLI UX contract:
  - [x] `kubecraft init --ip <x.x.x.x>` is required before `register`/server commands
  - [x] Runtime config stores: `clusterIP` (source), `tlsInsecure`, plus `username`/`token` (endpoints derived at runtime)
- [x] Decide strictness:
  - [x] Fail hard if cluster config missing (confirmed)
  - [ ] Allow command flags as one-off override (optional — deferred)
- [x] Freeze naming and YAML keys for config schema (to avoid churn later)
- [x] Write scope statement: "single public IP + fixed ports + NodePort model"
- [x] Document v1 cluster compatibility boundary
- **Exit criteria:** contract approved and implementation started

### Phase 1: Configuration Model Refactor

**Targets:** `internal/config/config.go`, `internal/config/constants.go`, `internal/config/*test.go`

- [x] Extend `~/.kubecraft/config` schema (minimal):
  - [x] `clusterIP` (user-provided IP)
  - [x] `tlsInsecure` (bool, persisted after probe)
  - [x] Keep `username` and `token`
  - [x] Derive at runtime (not persisted): `clusterEndpoint` (`https://<ip>:6443`), `registrationAddress` (`http://<ip>:30099`), `nodeAddress` (`<ip>`)
- [x] Split validation by command context:
  - [ ] `init` validation: IP format only (deferred to Phase 2)
  - [x] `register` validation: cluster settings required (`ValidateForRegister`)
  - [x] `server *` validation: cluster settings + username/token required (`ValidateForServer`)
- [x] Drop legacy compatibility requirements (explicitly accepted — no users yet)
- [x] Remove endpoint/IP/tls dependence from build-time constants:
  - [x] Stop treating `ClusterEndpoint`, `NodeAddress`, `TLSInsecure` as runtime source of truth
  - [x] Keep them as fallback defaults only
- [x] Update/add config tests for:
  - [x] load/save new schema
  - [x] validation failures and messages per command context
  - [x] derivation logic (IP → endpoint/address), including IPv6 safety
- **Exit criteria:** config package tests pass with new schema ✅

### Phase 2: Add `kubecraft init` Command

**Targets:** `internal/cli/init.go` (new), `internal/cli/root.go`

- [ ] Add `kubecraft init --ip <x.x.x.x>` command
  - [ ] Validate IP format (IPv4/IPv6)
  - [ ] Derive and store:
    - [ ] `clusterEndpoint` = `https://<ip>:6443`
    - [ ] `registrationAddress` = `http://<ip>:30099`
    - [ ] `nodeAddress` = `<ip>`
  - [ ] Implement TLS probe:
    - [ ] Attempt API connection with strict TLS to `https://<ip>:6443`
    - [ ] On TLS cert failure, retry with `InsecureSkipVerify`
    - [ ] Persist `tlsInsecure=true` on fallback
    - [ ] Print explicit security warning when fallback is persisted
  - [ ] Print concise "next steps" hint (`kubecraft register --username <name>`)
- [ ] Guardrails:
  - [ ] Friendly error if IP is missing or invalid
  - [ ] Friendly error if API is unreachable on `:6443`
  - [ ] Warning if registration endpoint unreachable on `:30099`
- [ ] Update CLI tests:
  - [ ] init command validation
  - [ ] TLS probe behavior (secure + fallback)
  - [ ] Config persistence after init
- **Exit criteria:** init writes complete usable config and reports deterministic next step

### Phase 3: Rewire Runtime Consumers

**Targets:** `internal/cli/register.go`, `internal/cli/root.go`, `internal/cli/server/create.go`, `internal/cli/server/start.go`, `internal/k8s/client.go`

- [ ] Update `register`:
  - [ ] Load cluster settings from user config
  - [ ] Build registration URL from `registrationAddress` + `/register`
  - [ ] Stop using build-time `config.ClusterEndpoint`
- [ ] Update root pre-run client creation:
  - [ ] Use `AppConfig.ClusterEndpoint` and `AppConfig.TLSInsecure`
  - [ ] Fail with clear message if init not run
- [ ] Update server ready output:
  - [ ] Use `AppConfig.NodeAddress` instead of `config.NodeAddress`
- [ ] Update K8s client constructor:
  - [ ] Accept runtime TLS setting parameter
  - [ ] Avoid global constant dependency for endpoint/TLS
- [ ] Update CLI tests:
  - [ ] register path construction from runtime config
  - [ ] missing-config errors
  - [ ] root client initialization behavior
- **Exit criteria:** user can run `init -> register -> server create/list/start/stop/delete` without any ldflag endpoint/IP

### Phase 4: Build Simplification

**Targets:** `Makefile`, docs mentioning build commands

- [ ] Remove `build-prod` target entirely
- [ ] Collapse to one build target (`build`) as the canonical command
- [ ] Keep build-time ldflags only for values that are safe/global:
  - [ ] `ServerImage` default tag (if desired)
  - [ ] Remove `ClusterEndpoint`, `NodeAddress`, `TLSInsecure` from ldflags
- [ ] Remove `PROD_ENDPOINT`, `PROD_NODE_ADDRESS`, and prod-only ldflags wiring
- [ ] Ensure dev workflow still works:
  - [ ] k3d users configure runtime endpoint via `kubecraft init`
  - [ ] Update dev defaults in Makefile if needed
- [ ] Update command map in `AGENTS.md`
- **Exit criteria:** only one supported build command remains and no infra-specific build inputs exist

### Phase 5: Release Pipeline Hardening

**Targets:** `.github/workflows/release.yml`

- [ ] Remove `vars.PROD_ENDPOINT` and `vars.PROD_NODE_ADDRESS` usage
- [ ] Remove endpoint/node/tls ldflags from release build step
- [ ] Keep semantic versioning and checksum generation unchanged
- [ ] Add verification step in workflow:
  - [ ] Scan built binaries/metadata for forbidden strings (`CHANGEME`, known IP/domain patterns if desired)
- [ ] Remove now-unused repo variables from GitHub settings
- **Exit criteria:** release artifacts are environment-agnostic and reproducible

### Phase 6: Documentation Migration

**Targets:** `README.md`, `AGENTS.md`, optional migration doc

- [ ] Rewrite onboarding:
  - [ ] Install CLI (generic artifact)
  - [ ] Install Helm chart on your cluster
  - [ ] Run `kubecraft init --ip <public-ip>`
  - [ ] Run `kubecraft register --username <name>`
  - [ ] Run server commands
- [ ] Remove "binary ships pre-configured" language from `README.md:68`
- [ ] Replace `make build-prod` references (`README.md:146`, `AGENTS.md:15`, related notes)
- [ ] Add scope statement:
  - [ ] v1 supports clusters where one public IP serves both API and NodePort traffic
  - [ ] Clear compatibility boundary
- [ ] Add troubleshooting section:
  - [ ] API unreachable on `:6443` → check security group / firewall
  - [ ] Registration unreachable on `:30099` → check Helm chart / NodePort service
  - [ ] Insecure TLS fallback warning → meaning and how to verify
- **Exit criteria:** docs describe only runtime configuration + single build command

### Phase 7: Validation Matrix

- [ ] Unit tests: `make test`
- [ ] Helm lint: `helm lint ./charts/kubecraft-control-plane`
- [ ] Integration tests with real cluster: `go test -p 1 -tags=integration ./internal/...`
- [ ] Manual smoke on clean machine profile:
  - [ ] Fresh install of generic binary
  - [ ] `kubecraft init --ip <cluster-ip>`
  - [ ] `kubecraft register --username <name>`
  - [ ] Server lifecycle commands (create/list/start/stop/delete)
- [ ] Release dry run from workflow branch:
  - [ ] Confirm checksums
  - [ ] Confirm no embedded prod endpoint/IP
- **Exit criteria:** all test layers green + smoke test complete

### Phase 8: HTTPS Migration Prep (Future)

- [ ] Add design doc for registration HTTPS endpoint:
  - [ ] Cert distribution strategy (let's encrypt, self-signed, or CA)
  - [ ] TLS config in registration service
  - [ ] CLI trust store handling
- [ ] Keep current HTTP path intact until TLS onboarding is ready
- [ ] Plan backward compatibility for existing HTTP registrations
- **Exit criteria:** approved migration design without breaking current one-IP UX

---

## Dependency Chain (Critical Path)

```
Phase 0 (Contract Freeze)
    ↓
Phase 1 (Config Refactor)
    ↓
Phase 2 (Init Command)
    ↓
Phase 3 (Rewire Consumers)
    ↓
Phase 4 (Build Simplification) ──┐
    ↓                             │
Phase 5 (Release Hardening) ◄───┘
    ↓
Phase 6 (Docs Migration)
    ↓
Phase 7 (Validation Matrix)
    ↓
Phase 8 (HTTPS Migration Prep) [decoupled]
```

**Parallelizable:**
- Phase 4 and Phase 5 can happen in parallel after Phase 3.
- Phase 6 can start as soon as Phase 3 is done (docs draft), finalized after Phase 4/5.
- Phase 8 is decoupled and can happen anytime after Phase 7.

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| TLS auto-fallback hides security issues | Medium | Always print explicit warning + persisted-state notice; user must see and acknowledge |
| Generic K8s ambiguity | Medium | Docs must clearly declare v1 network assumptions (one IP, fixed ports, NodePort) |
| Config-state drift | Low | Strict validation messages that point to `kubecraft init` when settings missing |
| Release pipeline still embeds secrets | High | Verification step scans binaries for forbidden strings; remove all env-specific ldflags |
| Existing tests break with config changes | Medium | Update all tests in Phase 1 before changing consumers; integration tests need cluster |
| User enters wrong IP | Medium | Validate IP format; probe API/registration and fail fast with clear error |

---

## Key Decision Rationale

- **Single build command (`build`)**: Removes the leaking public-vs-private artifact split. One binary for all users, regardless of environment.
- **Runtime config (`init`)**: Endpoint, node address, and TLS settings are cluster-specific. They belong in user state, not compile-time data.
- **No embedded endpoint/IP in public artifacts**: Guarantees the generic artifact can be redistributed safely without exposing on-prem infrastructure details.
- **Fixed ports (6443/30099)**: Keeps the one-input UX contract. Ports are predictable and documented. Override flags can be added later if needed.
- **Explicit `init` step**: Clear state transitions and easier troubleshooting vs. auto-discovery magic.

---

## Execution Order (Recommended)

| Wave | Phases | What Happens |
|------|--------|--------------|
| 1 | 0 → 1 → 2 → 3 | Refactor config model and wire runtime values into all commands |
| 2 | 4 → 5 | Simplify build system and harden release pipeline |
| 3 | 6 → 7 | Rewrite docs and run validation matrix |
| 4 | 8 | Design HTTPS migration path |

---

## Appendix: Config Schema

### New `~/.kubecraft/config` Schema

```yaml
clusterIP: "203.0.113.10"           # User-provided
tlsInsecure: true                   # Persisted after probe
username: "alice"
token: "eyJhbG..."
```

Derived at runtime (not persisted):
- `clusterEndpoint` = `https://<clusterIP>:6443`
- `registrationAddress` = `http://<clusterIP>:30099`
- `nodeAddress` = `<clusterIP>`

### Validation Requirements by Command

| Command | Required Fields |
|---------|----------------|
| `init` | None (creates config) |
| `register` | `clusterIP` |
| `server *` | `clusterIP`, `username`, `token` |

---

## Appendix: Error Messages

### Missing Init

```
$ kubecraft register --username alice
Error: cluster not initialized. Run `kubecraft init --ip <public-ip>` first.
```

### Invalid IP

```
$ kubecraft init --ip not-an-ip
Error: invalid IP address: "not-an-ip"
```

### API Unreachable

```
$ kubecraft init --ip 203.0.113.10
Error: cannot reach Kubernetes API at https://203.0.113.10:6443
       Check that port 6443 is open in your security group / firewall.
```

### TLS Fallback

```
$ kubecraft init --ip 203.0.113.10
Warning: TLS certificate verification failed. Falling back to insecure mode.
         This is persisted in ~/.kubecraft/config. To re-verify, delete config and re-run init.
Successfully initialized cluster config.
Next step: kubecraft register --username <name>
```
