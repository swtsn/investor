# ADR-013: Setup flow — BucketService, panel-model TUI, client-side pct validation

## Status
Accepted

## Context

Phase 5 adds the only missing write paths in the app: creating and editing Buckets and Allocations. The DB layer already has all primitives (`CreateBucket`, `UpdateBucket`, `UpsertAllocation`, `DeleteAllocation`). The decisions below cover the service, gRPC, and TUI layers.

## Decisions

### 1. New `BucketService` for writes

Bucket/allocation write operations go into a new `BucketService`, not `PoolService`. `PoolService` is a coherent read model; adding writes would change its character. The pattern mirrors the existing split: `BudgetService` owns budget writes, `DeploymentService` owns deployment writes.

### 2. Proto surface

Four new RPCs:
- `CreateBucket(name, type, target_pct)` → `Bucket`
- `UpdateBucket(id, name, target_pct)` → `Bucket` — `type` is omitted from the request; it is immutable after creation
- `UpsertAllocation(bucket_id, name, target_pct)` → `Allocation`
- `DeleteAllocation(id)` → empty

### 3. Bucket type is immutable

Bucket `type` (`flat` / `diversified`) is set at creation and cannot be changed. `UpdateBucket` does not accept a `type` field. This is enforced by omitting `type` from the update proto message.

### 4. Allocation name is immutable

`UpsertAllocation` matches on `(bucket_id, name)` and can only update `target_pct`. Renaming an allocation requires delete + recreate, which risks orphaning historical deployments. Phase 5 makes name immutable; a proper `UpdateAllocation(id, name, pct)` by-ID path is deferred to when needed.

### 5. Service-layer validation

Enforced server-side, not just in the TUI:
- Name non-empty on all create/update operations
- `target_pct` > 0 on all create/update operations
- `UpsertAllocation` rejects flat buckets (`ErrBucketIsFlat` → gRPC `InvalidArgument`)
- `DeleteAllocation` is blocked if any deployment references the allocation (`ErrAllocationHasDeployments` → gRPC `FailedPrecondition`)

### 6. Pct-sum validation is advisory, client-side only

The server does not enforce that bucket `target_pct` values sum to 100%, or that allocation `target_pct` values within a bucket sum to 100%. Saves are never blocked. The TUI computes the sums locally and surfaces a "needs attention" indicator when they are off.

### 7. TUI: setup is a persistent browsing surface

Setup is entered with `s` from the dashboard and stays resident until `esc` returns to dashboard — parallel to the Month view, not a transient action flow. Internally it has two levels: bucket list → allocation list for the selected bucket.

### 8. TUI: panel model for forms

Create/edit forms appear as an inline panel below the active list, replacing the help bar area. The list stays visible for context. `esc` dismisses the form; `enter` on the last field submits.

### 9. TUI: "needs attention" indicator

When a bucket's pct configuration is off (bucket `target_pct` values don't sum to 100%, or a diversified bucket's allocation `target_pct` values don't sum to 100%), the dashboard row and the setup view row both show a `!` prefix and a highlighted row style (lipgloss warning color). The setup view also shows the full numbers inline.

### 10. DeleteAllocation requires inline confirmation

Pressing `d` on an allocation in the setup view shows a one-line `delete "name"? [y/n]` prompt inline. No full-screen confirmation step.

## Consequences

- `BucketService` is wired into the gRPC handler constructor alongside `BudgetService`, `DeploymentService`, and `PoolService`.
- Two new domain errors: `ErrBucketIsFlat`, `ErrAllocationHasDeployments`.
- The dashboard view gains a per-bucket warning indicator; the setup view is a new TUI mode (`modeSetup`).
- Allocation rename is not supported in Phase 5. Backlogged.
