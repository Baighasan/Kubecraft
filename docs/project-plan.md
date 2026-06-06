# Kubecraft Project Plan

## Project Definition

Kubecraft makes it easy for technical homelab users to manage multiple Minecraft servers across a small Kubernetes cluster through a CLI.

This is primarily a learning project for Kubernetes, infrastructure-as-code, platform engineering, Go CLI development, release automation, and homelab operations. It is also intended to become a resume/interview-ready project and a tool that is useful for running Minecraft servers with friends.

Kubecraft is not trying to compete with hosted Minecraft platforms such as Aternos. It is intentionally niche: a self-hosted platform for people who already have, or want to learn to run, Kubernetes in a homelab.

## Motivation

The original motivation was to learn Kubernetes and adjacent platform/devops tooling by building something real instead of a toy example. Minecraft hosting is a good fit because it has a concrete user loop, persistent state, networking concerns, resource constraints, container images, and multi-user operational boundaries.

The project started with an Oracle Cloud K3s reference deployment in mind, but capacity constraints made the homelab path more relevant. OCI remains an optional reference path, not the core deployment target for `v1.0.0`.

The project is intentionally over-engineered compared to the simplest way to host Minecraft. That is acceptable because Kubernetes is the subject of the project, not an incidental implementation detail.

## Target User

The target user is a technical person with a homelab who wants to use Kubernetes to host and manage Minecraft servers.

The expected user can:

- Operate or access a K3s/homelab Kubernetes cluster.
- Install a Helm chart.
- Download and run a CLI binary.
- Expose NodePorts on a LAN or VPN.
- Understand basic Kubernetes/networking troubleshooting when the cluster environment is not standard.

Kubecraft is optimized for trusted users on a LAN or VPN. It is not designed for hostile public multi-tenancy.

## Current POC Status

The current proof of concept already includes the core platform shape:

- Go CLI entrypoint in `cmd/kubecraft`.
- In-cluster registration service entrypoint in `cmd/registration-server`.
- Helm-managed static control-plane resources in `charts/kubecraft-control-plane`.
- Runtime-created tenant and server resources in `internal/k8s` and `internal/registration`.
- CLI commands for `init`, `register`, and `server create/list/start/stop/delete`.
- Registration flow that creates per-user namespaces, RBAC, resource quota, and a ServiceAccount token.
- Minecraft servers represented by StatefulSets, NodePort Services, and PVC-backed world storage.
- CI workflows for tests, image builds, manifest validation, and release automation.
- Optional Terraform/OCI reference infrastructure.

The POC proves the end-to-end idea, but it does not yet meet the final `v1.0.0` scope. The remaining work is mostly about tightening the product boundary, improving release/install polish, supporting the agreed multi-server quota model, and adding a few high-value operational CLI commands.

## Version Goal

The project will call the finished personal-project release `v1.0.0`.

For this project, `v1.0.0` means complete, documented, demonstrable, and useful for the intended homelab use case. It does not mean production-grade public hosting or enterprise support.

All release artifacts should use the same version tag:

- CLI binaries.
- Minecraft server image.
- Registration service image.
- Packaged Helm chart.
- GitHub Release tag.

## Functional Requirements

`v1.0.0` must support the following user flow:

1. A cluster admin installs the Kubecraft control plane with Helm.
2. A user downloads the matching Kubecraft CLI binary.
3. A user initializes the CLI with a DNS name or IP address for the cluster host.
4. A user optionally configures a separate Minecraft host used for printed connection addresses.
5. A user registers a username.
6. A user creates a Minecraft server with the interactive wizard.
7. A user can also create a server directly with defaults.
8. A user can list their servers.
9. A user can describe a server and see connection/runtime details.
10. A user can view server logs, including tail/follow/since behavior.
11. A user can stop and start a server while preserving world data.
12. A user can delete a server, permanently deleting its world data after confirmation.
13. A user can connect to the Minecraft server using the printed host and NodePort.

Required CLI surface for `v1.0.0`:

- `kubecraft init --host <dns-or-ip> [--minecraft-host <dns-or-ip>]`
- `kubecraft register --username <name>`
- `kubecraft server create`
- `kubecraft server create <name>`
- `kubecraft server list`
- `kubecraft server describe <name>`
- `kubecraft server logs <name> [--tail N] [--follow] [--since duration]`
- `kubecraft server start <name>`
- `kubecraft server stop <name>`
- `kubecraft server delete <name>`

Required platform behavior:

- Multiple users can register on the same cluster.
- Each user can own up to 3 total servers by default.
- Stopped servers still count toward the 3-server quota because they keep PVCs and NodePorts.
- Server resources are static for `v1.0.0`, not user-selected per server.
- Tenant max servers and server CPU/memory/storage policy are configured by Helm.
- Max users remains hardcoded at 15 for `v1.0.0`.
- ServiceAccount token expiry is reduced from 5 years to 1 year.
- World data survives stop/start through PVC storage.
- Delete permanently removes the server and associated world data.

## Non-Functional Requirements

Kubecraft should be designed for small K3s/homelab clusters, with generic Kubernetes support on a best-effort basis.

`v1.0.0` non-functional requirements:

- Installation should not require building binaries or images locally.
- Release artifacts should be available through GitHub Releases and GHCR.
- The Helm chart should install the control plane using versioned images.
- The CLI should clearly print the Minecraft address a user needs to type into the Minecraft client.
- NodePort reachability should be documented as the cluster admin's responsibility.
- The project should pass unit tests, integration tests, Helm manifest checks, and CI workflows.
- The project should be manually validated on the author's homelab cluster with friends before being called done.
- Security posture should be honest: trusted LAN/VPN users, HTTP registration deferred, no public hostile tenancy claim.

## Explicitly Out Of Scope

The following are out of scope for `v1.0.0`:

- Web dashboard.
- User deletion or unregistration.
- Cross-user sharing, viewing, or controlling another user's servers.
- Minecraft console command execution.
- Kubernetes pod shell exec through Kubecraft.
- Backups, export, or restore.
- HTTPS registration endpoint.
- Admin portal or admin CLI.
- Dynamic Minecraft version fetching from remote APIs.
- Arbitrary user-selected CPU, memory, or storage per server.
- LoadBalancer, Ingress, or Gateway API support.
- Required Terraform/OCI deployment path.
- Production-grade auth, audit logging, or public multi-tenant hardening.
- Payment, billing, or commercial hosting workflows.
- Stretch goals before homelab validation.

These features may be considered after `v1.0.0`, but they should not block moving on from this project.

## Architecture

Kubecraft is split into four ownership areas:

- Helm owns static control-plane Kubernetes resources.
- Go runtime code owns dynamic tenant and server resources.
- Docker images own runtime packaging for the registration service and Minecraft server.
- Terraform is optional reference infrastructure and is not required for the core product.

High-level architecture:

```text
Technical User Machine
  kubecraft CLI
    |-- init/register over cluster host + registration NodePort
    |-- server lifecycle over Kubernetes API

Kubernetes Cluster
  kubecraft-system namespace
    Registration Service
      - validates registration requests
      - creates tenant namespaces and RBAC
      - issues ServiceAccount tokens

  tenant namespace: mc-<username>
    ServiceAccount + Role + RoleBinding
    ResourceQuota
    StatefulSet per Minecraft server
    NodePort Service per Minecraft server
    PVC per Minecraft server

Minecraft Client
  connects to <minecraft-host>:<nodePort>
```

The CLI stores local user configuration in `~/.kubecraft/config`. The config includes the cluster host, optional Minecraft host, TLS behavior, username, and token. The Kubernetes token is used by the CLI for server lifecycle operations within the user's namespace.

The registration service is the only component that needs cluster-wide write permissions. Normal users receive namespace-scoped permissions for their own resources.

## Networking Model

`v1.0.0` uses NodePort networking.

Kubecraft is responsible for creating NodePort Services and printing the configured Minecraft host with the allocated port. The cluster administrator is responsible for making that host and NodePort range reachable from Minecraft clients.

For homelab clusters, the Minecraft host might be:

- A node IP.
- A DNS name pointing at a node.
- A router-forwarded hostname.
- A LAN/VPN hostname.
- A VIP or other homelab-managed address.

Kubecraft will not auto-detect the best node, rotate between nodes, manage firewalls, or guarantee NodePort reachability on every Kubernetes distribution. Generic Kubernetes support is best-effort.

## Storage Model

Kubecraft does not provide a storage abstraction in `v1.0.0`.

Each server gets PVC-backed world storage. Stop/start preserves data. Delete removes the server and its data. If the cluster's StorageClass uses local persistent volumes, server data may be tied to the node where the volume lives. That is acceptable for `v1.0.0`.

Moving a server across nodes is only supported if the cluster's storage layer supports it. Kubecraft does not manage migration explicitly.

## Capacity And Quotas

Kubecraft uses a simple cluster-wide capacity check for `v1.0.0`.

This is intentionally simpler than scheduler-like node-aware capacity logic. The target environment is a trusted small homelab cluster, not a large multi-tenant production platform.

Policy decisions:

- 3 total servers per user by default.
- Static server resource profile.
- Tenant max servers and server CPU/memory/storage policy configured through Helm.
- Max registered users remains 15 for `v1.0.0`.
- NodePort range remains a documented cluster/networking assumption.

## Security Model

Kubecraft assumes trusted homelab users on a LAN or VPN.

Security decisions for `v1.0.0`:

- Registration remains HTTP and is intended for trusted networks.
- Registration is open to users who can reach the registration NodePort, up to the max user limit.
- Tokens are Kubernetes ServiceAccount tokens scoped to tenant namespaces.
- Token expiry should be 1 year.
- Normal user permissions are namespace-scoped.
- Cluster-wide write permissions are limited to the registration service.

This is good enough for a homelab learning project, but it is not appropriate for public untrusted hosting. Public-hosting hardening, HTTPS registration, invite tokens, admin-managed users, audit logging, and token rotation are post-`v1.0.0` concerns.

## Key Tradeoffs

### Kubernetes vs Simpler Hosting

The simplest Minecraft hosting tool would not use Kubernetes. Kubecraft does because Kubernetes is the learning target and the product domain. Namespaces, RBAC, StatefulSets, PVCs, Services, Helm, and cluster networking are the point of the project.

### CLI vs Web Dashboard

A CLI is faster to build, easier to test, and better aligned with technical homelab users. A dashboard would add frontend, auth, sessions, and UI maintenance without improving the core learning goals enough for `v1.0.0`.

### NodePort vs LoadBalancer/Ingress

NodePort is simple and works well enough for LAN/VPN homelab clusters. LoadBalancer, Ingress, and Gateway API support would require more cluster-specific networking decisions. Those are deferred.

### Static Server Resources vs User Tuning

Static resources keep quota, capacity checks, and support docs simple. Per-server resource tuning is useful, but it adds policy and validation complexity that is not needed for the first complete release.

### Trusted Users vs Hostile Tenants

Designing for trusted friends keeps the project finishable. Hostile tenancy would require a higher security bar, stronger registration controls, auditability, and probably a different administration model.

### No Backups Before `v1.0.0`

Backups matter, but they expand the operational scope significantly. For `v1.0.0`, PVC persistence across stop/start is required, while backups/export/restore are deferred.

## Milestones To Finish `v1.0.0`

1. Replace `kubecraft init --ip` with DNS-capable `kubecraft init --host`.
2. Add optional `--minecraft-host` and persist it in CLI config.
3. Print Minecraft connection addresses from `minecraftHost` plus allocated NodePort.
4. Support 3 total servers per user through tenant quota changes.
5. Make tenant max servers and server CPU/memory/storage configurable through Helm-controlled policy.
6. Reduce ServiceAccount token expiry from 5 years to 1 year.
7. Add `kubecraft server describe <name>`.
8. Add `kubecraft server logs <name> [--tail N] [--follow] [--since duration]`.
9. Package the Helm chart as a GitHub Release artifact.
10. Ensure CLI binaries, images, and chart versions all match `v1.0.0`.
11. Update README install, release, and troubleshooting docs.
12. Validate unit tests, integration tests, Helm checks, and CI workflows.
13. Deploy to the author's homelab cluster.
14. Use Kubecraft with friends at least once.
15. Publish the `v1.0.0` release.

No stretch goals are allowed before these milestones are complete.

## Release Criteria

`v1.0.0` can be released when all of the following are true:

- CLI binaries are built and attached to a GitHub Release.
- Minecraft and registration images are published to GHCR with the `v1.0.0` tag.
- The Helm chart is packaged and attached to the GitHub Release.
- Checksums are generated for release binaries.
- Unit tests pass.
- Integration tests pass in k3d or an equivalent real Kubernetes test environment.
- Helm chart lint/template validation passes.
- CI workflows are green.
- The README explains install, initialization, registration, server lifecycle, and troubleshooting.
- This project plan explains the architecture, tradeoffs, scope, and non-goals.
- The project has been deployed on the author's homelab cluster and used with friends once.

## Done Definition

Kubecraft is done when a technical homelab user can install the Helm chart on a K3s or generic Kubernetes cluster, download the CLI, initialize with a reachable host, register, create up to three Minecraft servers, connect through the printed Minecraft address, inspect status and logs, stop/start servers while preserving world data, delete servers, and understand the architecture/tradeoffs from the project documentation.

At that point, the project is complete enough to move on.

## Post-`v1.0.0` Roadmap

Possible future work, explicitly after `v1.0.0`:

- User deletion or unregistration.
- Cross-user sharing for viewing or starting/stopping shared servers.
- Backups, export, and restore.
- HTTPS registration or VPN-first hardening docs.
- Admin-managed registration or invite token.
- LoadBalancer support.
- Better multi-node networking/address detection.
- Optional Terraform/OCI validation.
- Dynamic Minecraft version discovery.
- Resource presets or per-server resource tuning.

These are not required to call the project complete.

## Maintenance Policy

Kubecraft will be maintained while the author personally uses it. It is a personal homelab project with no production support guarantee.

If it stops being personally useful, it is acceptable to stop maintaining it, archive it, or leave it as a completed learning/recruiting artifact.
