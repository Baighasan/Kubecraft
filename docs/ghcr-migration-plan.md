# Kubecraft GHCR Migration Plan

## Overview

This plan migrates all container images from Docker Hub (`hasanbaig786/...`) to GitHub Container Registry (`ghcr.io`). It introduces an automatic release system with semantic versioning and establishes a dedicated GitHub Release pipeline for the CLI binary.

**Scope:** Public packages only. No `imagePullSecrets` required.

**Three Deliverables:**
1. `ghcr.io/<owner>/kubecraft-minecraft` — Minecraft server container image
2. `ghcr.io/<owner>/kubecraft-registration` — Registration service container image
3. `kubecraft` CLI binary — Distributed via GitHub Release assets (not a GHCR package)

---

## Release Strategy

### Versioning Policy

We use standard **three-segment** semantic versioning: `vMAJOR.MINOR.PATCH` (e.g., `v1.9.0`, `v2.0.0`).

- **Patch bump (default):** Every release-labelled PR merge to `main` bumps patch by default.
- **Minor bump:** When the merged PR has the label `semver:minor`.
- **Major bump:** Only when the merged PR has the label `semver:major`.

### Release Trigger

Releases are **opt-in** via PR label:

1. PR must have the `release` label to trigger a release on merge.
2. If `semver:major` present → bump major (e.g., `v1.9.0` → `v2.0.0`).
3. If `semver:minor` present → bump minor (e.g., `v1.9.0` → `v1.10.0`).
4. Otherwise → bump patch (e.g., `v1.9.0` → `v1.9.1`).
5. Create Git tag `vX.Y.Z`.
6. Publish GHCR images with release tags.
7. Create GitHub Release with CLI binaries.

**Why opt-in?**
This prevents noise from trivial merges (docs fixes, refactors) and keeps release history meaningful. Only PRs you explicitly mark as release-worthy cut a new version.

---

## Tagging Strategy

| Event | Tag Produced | Example |
|-------|-------------|---------|
| Push to `dev` (non-`main`) branch | `dev` | `ghcr.io/.../kubecraft-minecraft:dev` |
| Pull Request | Build only (no push) | — |
| Git tag push `v*` (from release workflow) | Exact version | `ghcr.io/.../kubecraft-minecraft:v0.1.0` |

**Rules:**
- `:dev` is a **mutable** tag used only for active development on the `dev` branch. It is overwritten on every push.
- Production deployments should always pin to a release tag (`vX.Y.Z`).
- There is no `latest` tag. Use explicit versions only.

**Tag Examples:**

Push to `dev` branch:
- `ghcr.io/baighasan/kubecraft-minecraft:dev`

If latest release is `v0.1.0` and a `release`-labelled PR merges without `semver:major` or `semver:minor`:
- New release: `v0.1.1`
- Image tag published:
  - `ghcr.io/baighasan/kubecraft-minecraft:v0.1.1`
  - (same pattern for registration image)
- CLI: GitHub Release `v0.1.1` with binaries and checksums

If PR has `semver:minor`:
- `v0.1.0` → `v0.2.0`

If PR has `semver:major`:
- `v0.1.0` → `v1.0.0`

---

## Phase 1: Registry Preparation

**Goal:** Ensure GHCR is ready to receive public packages.

**Steps:**
1. In GitHub repo settings, confirm Actions has **read/write access to packages**.
2. After the first workflow run, navigate to each GHCR package page and set visibility to **Public**.
3. Document the canonical package names (replace `<owner>` with your GitHub username/org in lowercase):
   - `ghcr.io/baighasan/kubecraft-minecraft`
   - `ghcr.io/baighasan/kubecraft-registration`

**Files affected:** None (config-only).

---

## Phase 2: Minecraft Image Workflow

**Goal:** Publish the Minecraft server image to GHCR with controlled tags.

**File:** `.github/workflows/minecraft-image.yml`

**Changes:**
- Add job-level `permissions`:
  ```yaml
  permissions:
    contents: read
    packages: write
  ```
- Add `docker/login-action@v3` to authenticate to GHCR using `GITHUB_TOKEN`.
- Replace hardcoded `tags: hasanbaig786/kubecraft:latest` with `docker/metadata-action@v5` to generate tags dynamically based on branch/tag context.
- **Push to `dev` branch:** Build and push `:dev` tag (mutable, overwritten every push).
- **Push `v*` tag (from release workflow):** Push exact version tag (`vX.Y.Z`).
- **PRs:** Build only (`push: false`).
- Build for `linux/amd64` only.

---

## Phase 3: Registration Image Workflow

**Goal:** Publish the registration service image to GHCR with the same tagging discipline.

**File:** `.github/workflows/registration-image.yml`

**Changes:**
- Apply the exact same pattern as Phase 2 (permissions, GHCR login, metadata-action, conditional push).
- Target Dockerfile: `docker/registration/Dockerfile`.
- Package name: `ghcr.io/baighasan/kubecraft-registration`.
- Build for `linux/amd64` only.

---

## Phase 4: Auto-Release Workflow

**Goal:** Automatically create releases when a PR with the `release` label is merged to `main`.

**File:** `.github/workflows/release.yml` (new)

**Trigger:** `on: pull_request: types: [closed]` with condition `if: github.event.pull_request.merged == true && github.event.pull_request.base.ref == 'main' && contains(github.event.pull_request.labels.*.name, 'release')`

### Why This Trigger Instead of `push` to `main`?

When a PR is merged, the `push` event on `main` fires **but the push payload does not contain the merged PR's labels**. GitHub Actions `push` events only know about the commit SHA and message — they have no direct link back to the PR that was merged.

**The Problem:**
If you use `on: push: branches: [main]`, your workflow cannot directly determine:
- Whether the merge came from a PR (vs. a direct push)
- What labels that PR had
- Whether it should trigger a release

You would have to call the GitHub API to search for PRs associated with the merge commit SHA, which is fragile and race-prone.

**The Solution:**
Use `on: pull_request: types: [closed]` instead. When a PR is closed (including merged), the workflow receives the full PR object in `github.event.pull_request`, including:
- `merged` — boolean, true if merged
- `base.ref` — the target branch (`main`)
- `labels` — array of label objects

This allows reliable, direct access to the `release` and `semver:*` labels without API calls.

**Workflow Steps:**
1. **Check labels:** Verify `release` label is present (the trigger condition handles this, but double-check in job logic).
2. **Determine bump type:**
   - `semver:major` → major bump
   - `semver:minor` → minor bump
   - Otherwise → patch bump (default)
3. **Compute next version:** Query latest Git tag, apply bump (e.g., `v1.9.0` → `v1.9.1`).
4. **Create Git tag:** Push `vX.Y.Z` tag to the repo using `actions/github-script` or a git push step.
5. **Build CLI binaries:**
   - Matrix: `linux/darwin/windows` × `amd64/arm64`
   - Use `go build` with `ldflags` for prod endpoint config.
   - Produce `kubecraft-<os>-<arch>` (or `.exe` for Windows).
   - Generate `checksums.txt`.
6. **Create GitHub Release:** Use `softprops/action-gh-release` to create release from the new tag, attach binaries and checksums.
7. **Trigger image release tags:** Call `workflow_dispatch` on the image workflows or tag the commit directly so the image workflows pick up the version tag.

**Label Requirements for PRs:**
- `release` — Required to trigger a release on merge.
- `semver:major` — Optional; bumps major version.
- `semver:minor` — Optional; bumps minor version.
- No semver label — Default to patch bump.

---

## Phase 5: Runtime Reference Migration

**Goal:** Update all in-repo references so they resolve to GHCR instead of Docker Hub, and add dev override capability.

### 5.1 Minecraft Server Image (Go variable)
**File:** `internal/config/constants.go`
- Change `ServerImage` from `const` to `var` so it can be overridden at build time via `ldflags`:
  ```go
  var ServerImage = "ghcr.io/baighasan/kubecraft-minecraft:latest"
  ```
- Dev builds default to `:dev` via `Makefile` ldflags.
- Release builds pin to the exact release tag (`vX.Y.Z`).

### 5.2 Registration Service Image (Helm values)
**File:** `charts/kubecraft-control-plane/values.yaml` (lines 8–9)
- Change:
  ```yaml
  image:
    repository: ghcr.io/baighasan/kubecraft-registration
    tag: latest
    pullPolicy: IfNotPresent
  ```
- Production deployments should override at install time:
  ```bash
  helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane \
    --set registration.image.tag=v0.1.0
  ```
- Dev deployments use `:dev`:
  ```bash
  helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane \
    --set registration.image.tag=dev
  ```

### 5.3 CLI `--server-image` Flag
**File:** CLI command definitions (Cobra commands in `internal/cli/server/`)
- Add `--server-image` flag and `KUBECRAFT_SERVER_IMAGE` env var for runtime override.
- The CLI falls back to the build-time `config.ServerImage` when no override is provided.
- Example:
  ```bash
  kubecraft server create myserver --server-image=ghcr.io/baighasan/kubecraft-minecraft:dev
  # or via env:
  KUBECRAFT_SERVER_IMAGE=ghcr.io/baighasan/kubecraft-minecraft:dev kubecraft server create myserver
  ```

---

## Phase 6: Dev Flow & Local Testing

**Goal:** Enable fast iteration when working on images.

### Option A: Local Build + k3d Import (Fastest)
For tight development loops without pushing to remote:
```bash
# Build locally
docker build -t kubecraft-minecraft:dev -f docker/minecraft/Dockerfile .

# Import into k3d cluster
k3d image import kubecraft-minecraft:dev -c kubecraft-dev

# Deploy with local tag
helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane \
  --set registration.image.tag=dev

# Create server using local image
kubecraft server create myserver --server-image=kubecraft-minecraft:dev
```

### Option B: Use Remote `:dev` Tags
1. Push to `dev` branch.
2. CI builds and publishes:
   - `ghcr.io/baighasan/kubecraft-minecraft:dev`
   - `ghcr.io/baighasan/kubecraft-registration:dev`
3. Build the dev CLI locally (`make build-dev` — already defaults to `:dev`).
4. Install control-plane with `:dev` tag:
   ```bash
   helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane \
     --set registration.image.tag=dev
   ```
5. Create server — the dev binary already points to `:dev`:
   ```bash
   kubecraft server create myserver
   ```

**Recommendation:** Use Option A for the tightest iteration loop. Use Option B when you want CI to validate the build before you test it locally.

---

## Phase 7: Deployment & Validation

**Goal:** Verify the new images work end-to-end before relying on them in production.

### 7.1 Local k3d Validation
1. Push to `dev` branch and wait for CI to publish `:dev` tags.
2. Install the control-plane chart with the `:dev` tag:
   ```bash
   helm upgrade --install kubecraft-control-plane ./charts/kubecraft-control-plane \
     --set registration.image.tag=dev
   ```
3. Verify the registration pod is `Running` and events show a successful GHCR pull.
4. Build the CLI locally (`make build-dev` — defaults to `:dev`).
5. Run `kubecraft register --username testuser`.
6. Run `kubecraft server create testsvr`.
7. Inspect the resulting StatefulSet and confirm the container image is `ghcr.io/baighasan/kubecraft-minecraft:dev`.
8. Confirm the Minecraft pod starts and the readiness probe passes.

### 7.2 Integration Tests
- Run the existing integration suite against the k3d cluster:
  ```bash
  make test-all
  ```
- All tests must pass with the new GHCR-based defaults.

### 7.3 OCI Production Rollout
- Update your production Helm install command to point to the desired GHCR tag.
- Ensure the OCI node can reach `ghcr.io` (no firewall blocks).
- Because the images are public, no `imagePullSecrets` are required.

---

## Phase 8: Documentation Updates

**Goal:** Ensure contributors and users know where images live and how to pin them.

**Files to update:**
- `README.md`
  - Replace any Docker Hub references with GHCR.
  - Document the three deliverables and their locations.
  - Add a "Tagging Policy" subsection.
  - Show example Helm install with a pinned tag.
  - Add a "Download CLI" section pointing to GitHub Releases.
  - Add a "Dev Flow" section covering local build vs preview tags.
- `TESTING.md`
  - Update any commands that imply old image names.
- `AGENTS.md`
  - If image names or build processes are referenced, update them to match.

---

## Rollback & Risk Mitigation

| Risk | Mitigation |
|------|------------|
| GHCR unavailable during early rollout | Keep Docker Hub images for 30 days. Roll back via Helm `--set registration.image.repository=hasanbaig786/kubecraft-registration` or by reverting the Go constant and rebuilding the CLI. |
| `:dev` drift | `:dev` is intentionally mutable and only for local dev. Never use `:dev` in production. |
| amd64-only images on ARM64 node | This project targets amd64 hosting. If ARM64 is needed later, add multi-arch as a follow-up. |
| CLI release build break | Test the release workflow on a pre-release branch before first real merge to `main`. |
| Major/minor bump by accident | Require `semver:major` or `semver:minor` label on PR; default is always patch. Review PR labels before merge. |
| Release label forgotten | Add a PR template with a checklist reminding authors to add `release` label if the change is release-worthy. |

---

## Success Criteria

- [ ] `.github/workflows/minecraft-image.yml` pushes `:dev` on branch pushes and exact `vX.Y.Z` on tag pushes.
- [ ] `.github/workflows/registration-image.yml` pushes `:dev` on branch pushes and exact `vX.Y.Z` on tag pushes.
- [ ] `.github/workflows/release.yml` triggers only on merged PRs to `main` with the `release` label.
- [ ] `.github/workflows/release.yml` creates GitHub Releases with OS/arch binaries and checksums on every qualifying merge.
- [ ] `internal/config/constants.go` references the GHCR Minecraft image.
- [ ] `charts/kubecraft-control-plane/values.yaml` references the GHCR registration image.
- [ ] CLI supports `--server-image` flag for dev overrides.
- [ ] k3d integration tests pass using GHCR-based defaults.
- [ ] `:dev` tag is published on every push to the `dev` branch.
- [ ] README documents the new registry locations, tagging policy, dev flow, and CLI download instructions.

---

## Timeline Estimate

- Phase 1 (Registry prep): 10 min
- Phase 2 & 3 (Image workflows): 1–2 hours
- Phase 4 (Auto-release workflow): 1–1.5 hours
- Phase 5 (Runtime ref migration + CLI flag): 1 hour
- Phase 6 (Dev flow setup): 30 min
- Phase 7 (Validation): 1–2 hours
- Phase 8 (Docs): 30 min

**Total:** ~1.5–2 dev days.
