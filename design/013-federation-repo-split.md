```yaml
---
title: Federation Controllers & Server Repository Split
version: v1alpha1
authors: ESO Maintainers
creation-date: 2025-11-19
status: draft
---
```

# Federation Controllers & Server Repository Split

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary
The enterprise distribution currently bundles the federation controllers and the HTTP/mTLS federation server into the primary controller binary. This document proposes relocating all federation-specific code, manifests, and release automation into a dedicated repository and binary. The new repo will own the lifecycle of federation CRDs, controllers, APIs, images, and Helm assets while the core repo retains only the interfaces they depend on. The split reduces blast radius, accelerates federation releases, and enables independent scaling/deployment models.

## Motivation
* **Operational isolation**: Federation workloads have unique scaling and security requirements (exposed HTTP endpoints, TLS, SPIFFE integration). Splitting them allows independent resource limits, ingress, and deployment cadence.
* **Release agility**: Federation features can iterate without blocking on core controller releases and without bringing unrelated code into clusters for customers who do not use federation.
* **Codebase clarity**: The primary controller codebase becomes leaner and easier to reason about, preventing federation-only dependencies from leaking into the general operator path.
* **Security & compliance**: Sensitive federation logic (authorization, credential issuance) can be versioned, audited, and patched separately, reducing dependency conflicts.

### Goals
1. Deliver a dedicated repository (temporary name: `external-secrets-federation`) that builds, tests, and ships the federation server/controllers plus CRDs and Helm manifests.
2. Provide stable, versioned interfaces so the new repo can reuse required APIs/utilities from the main repo without circular dependencies.
3. Offer a migration path that lets operators adopt the standalone federation deployment with minimal downtime.
4. Maintain feature parity: no federation functionality regresses when relocated.

### Non-Goals
* Changing federation APIs or semantics (AuthorizedIdentity, FederationRef, etc.).
* Rewriting controller-runtime foundations.
* Altering existing customer workloads beyond deployment topology changes.
* Refactoring unrelated enterprise components (workflow API, scan controllers, etc.).

## Proposal

### Current State Overview
* `cmd/controller/root.go` wires federation controllers (`AuthorizationController`, `KubernetesFederationController`, etc.) and launches `pkg/enterprise/federation/server` alongside the core operator.
* Federation controllers depend on:
  * Kubernetes schemes for federation CRDs (`fedv1alpha1`, `identity/v1alpha1`).
  * `secretstore.Manager`, generator resolvers, scheduler bootstrap, metrics, and feature flags defined in the core repo.
* Helm charts deploy a single controller Deployment exposing both operator and federation endpoints under one image.
* Additional federation assets currently live in this repo and must be addressed during the split:
  * Helm bits under `deploy/charts/external-secrets` (federation service, ports, RBAC).
  * Sample manifests and scripts under `config/samples/federation/**` (kind deploy scripts, client examples).
  * Generator registration for the federation generator in `pkg/enterprise/generator/register/register_esi.go` and its implementation in `pkg/enterprise/generator/federation`.
  * CRDs under `config/crds/bases/*federation*` and bundled `deploy/crds/bundle.yaml`.

### Target State
* New repository hosts:
  * Federation CRDs + identity CRDs and their generated manifests.
  * Controllers and server packages currently under `pkg/enterprise/federation/**` and `pkg/enterprise/controllers/federation/**`.
  * A dedicated controller-runtime binary (`cmd/federation/main.go`) that registers only federation schemes/controllers and starts the HTTP/mTLS server.
  * Container build assets (Dockerfile, goreleaser config) and Helm/Kustomize manifests for the federation deployment.
* Main repository remains the source of truth for shared APIs (ExternalSecret, Generator, core utilities). Exported interfaces encapsulate what federation needs (client builder, metrics helpers, secretstore manager, logging setup, feature flags).
* Enterprise Helm chart gains an optional dependency (or sub-chart) that deploys the external federation service.

### Implementation Phases

#### Phase 0 – Seams & Interfaces (current repo)
* ✅ Define narrow interfaces for federation dependencies: `pkg/enterprise/federation/deps.ExternalSecretAccessor` now encapsulates the ExternalSecret reconciler functions (`RuntimeClient`, `RuntimeScheme`, controller class, flood-gate flag). The federation server consumes only this contract and no longer depends on the concrete reconciler.
* ⏳ Produce a dependency inventory of federation usage (controllers, server, store/auth helpers, scheduler hooks, metrics) and wrap the remaining touchpoints in interfaces under `pkg/enterprise/federation/deps` or a shared package.
* ⏳ Move reusable helpers (metrics, cache toggles, secretstore manager builder) into subpackages intended for reuse.
* ⏳ Add build tags or wiring toggles so federation code paths can be disabled while still compiling, and wire CI to run `go test ./...` with federation disabled.
* Exit criteria (measurable):
  * All federation packages compile using only interface seams from `deps` and shared helpers (no direct imports of core reconcilers/controllers).
  * CI job proves: (a) normal build/tests pass, (b) build/tests with federation disabled via tag/flag also pass.
  * Dependency inventory document lists remaining federation dependencies with owners and planned abstraction locations.
  * Decision documented for components that span repos (e.g., whether the federation generator stays in core or moves).

##### Phase 0 Progress (seams in-tree)
* Added `deps.ExternalSecretAccessor` and switched the federation server to depend on it (no concrete controller imports).
* Added federation dependency seams: `SecretStoreManagerFactory` and `GeneratorResolver` in `pkg/enterprise/federation/deps` plus `server.WithDependencies` to inject alternatives during the split.
* Introduced `--enable-federation` flag (defaulted via `disable_federation` build tag) to disable wiring in `cmd/controller/root.go`; `make test.no-federation` runs `go test` with the tag for CI coverage.
* Decision: keep the federation generator registered in core for now so ExternalSecret consumers can still call the external federation service without depending on the new repo. Re-evaluate post-bootstrap when a cross-repo generator registration hook exists.

##### Federation Dependency Inventory (owners + abstraction plan)
| Touchpoint | Dependency surface | Owner | Abstraction/plan |
| --- | --- | --- | --- |
| Federation server (generate/revoke/get) | Controller client/scheme, controller-class/floodgate settings, secretstore manager, generator resolver, ctrl logger | Federation server | Use `ExternalSecretAccessor` + new `SecretStoreManagerFactory` and `GeneratorResolver` seams (`server.WithDependencies`) so the new repo can inject replacements without importing core reconcilers. |
| Authorization / Federation controllers | controller-runtime manager (client/scheme), `store` cache, TLS allow-list in server, ctrl logger | Federation controllers | Move controllers plus `store`/TLS helpers to the new repo; no direct core-controller imports remain. |
| AuthorizedIdentity reconciliation | Same as above plus label selectors on GeneratorState | Federation controllers | Keep using structured client only; no extra seams needed once controllers move wholesale. |
| Federation generator (`pkg/enterprise/generator/federation`) | Generator registration, `resolvers.SecretKeyRef`, HTTP client | ExternalSecret/generator owners | Stays in core short-term; document handshake needed to move (new repo would export registration helper or module dependency). |
| SecretStore/provider clients for `/secretstore/...` endpoint | `secretstore.Manager` to resolve ClusterSecretStore, floodgate behavior | Store/targets owners | Abstracted via `SecretStoreManagerFactory`; default impl stays in core, new repo can carry a lightweight copy or adaptor. |

##### Relocation Plan for repo-split assets (pre-Phase 1)
* Helm: move federation service/port/RBAC bits from `deploy/charts/external-secrets` into the new repo’s chart; leave a thin values block in the core chart that depends on the external chart.
* Samples/scripts: move `config/samples/federation/**` (kind deploys, client examples) to the new repo; keep pointers in core README until overlap window ends.
* CRDs: move `config/crds/bases/*federation*` and `deploy/crds/bundle.yaml` generation to the new repo; core bundle drops federation after the overlap release.
* Generator registration: keep `pkg/enterprise/generator/federation` registered in core until a shared generator-registration mechanism exists; new repo will track the decision and own the docs/validation around the generator endpoint.

#### Phase 1 – Federation Repository Bootstrap
* Create new repo with `go.mod` that depends on a tagged main-repo version for shared APIs (no permanent `replace` hacks).
* Copy federation CRDs, controllers, server code, store/auth packages, and associated tests; ensure imports point to module versions, not relative paths.
* Decide and document whether identity CRDs move with federation or stay in core; reflect that choice in go.mod and manifests.
* Implement `cmd/federation/main.go` that instantiates a controller-runtime manager, registers the copied controllers, builds the HTTP/mTLS server, and exposes equivalent CLI flags.
* Ensure `go test ./...` passes in isolation; generate CRDs in-repo (e.g., `config/crds`).
* Relocate sample assets (current `config/samples/federation/**`) into the new repo or delete/replace with pointers to the new location.
* Exit criteria (measurable):
  * `go test ./...` green in the new repo without `replace` pointing to local paths.
  * CRD generation succeeds and produces artifacts under version control.
  * Imports from core resolve via module version; a static check (e.g., `rg ../external-secrets`) returns empty.
  * README/bootstrap doc explains how to build and run the standalone binary.
  * Sample manifests/scripts for federation live in (and are exercised from) the new repo; none remain stale in core.

#### Phase 2 – Packaging & Release Tooling
* Add Dockerfile, Makefile targets, and `.goreleaser.yaml` to build/publish the federation binary and container images.
* Generate CRD bundles within the new repo (`config/crds`), mirroring existing scripts.
* Author Helm manifests (Deployment, Service, ServiceAccount, RBAC, PodMonitor) parameterized by the same flags/values as today; include values for TLS/SPFFE sockets.
* Wire CI pipelines to run unit tests, linting, SBOM/provenance (e.g., cosign attest), and publish container images/Helm charts.
* Exit criteria (measurable):
  * CI produces a signed image and chart artifact for a tagged build; SBOM/provenance artifacts are published.
  * Helm install of the chart with default values succeeds in a smoke environment (e.g., kind), verified in CI.
  * Make/CI targets documented (`make test`, `make docker-build`, `make helm-package`).

#### Phase 3 – Integration with Core Distribution
* Update enterprise Helm chart (or documentation) to reference the new federation chart as an optional component, defaulting to disabled for one release.
* Provide sample values demonstrating how to direct clients to the new service (ports, TLS certs, SPIFFE socket path) and the minimum core version the chart supports.
* Build an end-to-end test scenario (GitHub Action or Make target) that deploys both repos together on kind to validate interoperability and enforces the compatibility matrix.
* Exit criteria (measurable):
  * Core chart includes an optional dependency/value block for federation with defaults off and documented min core version.
  * CI e2e/contract test stands up core + federation charts on kind and exercises at least one federation flow (generate/revoke) successfully.
  * Published compatibility matrix (core vs federation versions) lives in docs and is verified by CI job.

#### Phase 4 – Cutover & Cleanup
* Release coordinated versions (e.g., main `vX.Y.0` plus federation `vX.Y.0`). Document upgrade steps, including enabling the new deployment and disabling in-tree federation flags.
* Remove federation wiring from `cmd/controller/root.go` after at least one minor release of overlap; emit log warnings if federation flags are still set in the old binary during the overlap window.
* Archive or delete federation-specific manifests remaining in the main repo.
* Exit criteria (measurable):
  * Federation controllers/server are no longer started by the core binary; startup warns if deprecated flags are present.
  * Federation manifests are removed from the core repo; the sole deployment path is via the external chart.
  * Upgrade guide published with rollback steps and overlap duration stated (e.g., one minor release).

### Migration & Rollout Considerations
* **Rollout**: Deploy the new federation service alongside the existing controller, switch traffic (ingress/service endpoints), then disable federation flags in the old deployment.
* **Rollback**: Keep old flags working for one release window. Operators can revert by re-enabling federation in the main deployment if the external service fails.
* **Compatibility**: Version the federation repo in lockstep with core releases initially (e.g., require `controller v0.10.x` with `federation v0.10.x`), then relax once API surface stabilizes.
* **Telemetry**: Ensure metrics/health endpoints remain consistent; expose new PodMonitor resources.

### Drawbacks
* Maintaining two repositories increases coordination overhead (issues, releases, security fixes).
* Cross-repo interfaces risk divergence; requires diligent semantic versioning and contract tests.
* Users must deploy an additional workload, slightly increasing footprint.

### Acceptance Criteria
* Federation repo builds, tests, and publishes images/Helm charts independently.
* Enterprise Helm chart (or docs) instructs users how to deploy the standalone federation service; CI verifies the path.
* Old controller binary no longer ships federation controllers/server by default, with documented migration path.
* Version compatibility matrix published and validated via automated tests.
* Observability parity: metrics, health probes, and logging available in the new deployment.

## Alternatives
1. **Split within same repo**: Keep all code but build two binaries. Simpler but doesn’t reduce repo size or enable independent release cadence.
2. **Monorepo with packages**: Move federation into submodules instead of a new repo. Reduces Git churn but still forces synchronized releases.
3. **Feature flags only**: Keep code co-located and rely on operator flags to disable federation. Lowest effort but fails to achieve isolation/operator concerns.

The dedicated repository offers the clearest separation and aligns with enterprise packaging goals, despite the overhead of multi-repo management.
