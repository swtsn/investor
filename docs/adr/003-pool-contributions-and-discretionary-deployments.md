# ADR-003: Flat contribution pool with origination tags

## Status
Accepted (amended — `DeploymentType` removed; see amendment below)

## Context

Money enters a bucket from multiple sources: monthly budget allocation and reinvestment of realized gains. Undeployed money accumulates as slush over time.

## Decision

- All money entering a pool is recorded as a **Contribution** with an origination tag: `budget`, `reinvestment`, or `slush`.
- Slush is an explicit `Contribution` with `origination=slush`, auto-generated per bucket on the *first* `BudgetEvent` of a new month (top-up budget events in the same month do not generate slush). It is not derived at read time.
- `slush_amount = Σ all-time contributions − Σ all-time deployments` for the bucket at the moment the first budget event of the month is applied. Only written if `slush_amount > 0`.
- The UI surfaces origination breakdown — new budget, reinvestment, and carry-over slush — as a simple group-by over origination types within the current month.

## Consequences

Slush is queryable and auditable without arithmetic over unbounded history. `PoolBreakdown` is a plain group-by with no special derivation logic. The trade-off is that slush must be written explicitly on the first `ApplyBudget` call of a month — it is not self-correcting if a prior month's data is edited retroactively.

## Amendment: `DeploymentType` removed (Phase 1 design)

The original decision included a `DeploymentType` enum (`allocated | discretionary`) on `Deployment`. This was removed during Phase 1 domain modelling. The distinction is now expressed structurally: a diversified-bucket deployment with `AllocationID = nil` is an experimental/bypass deployment; a flat-bucket deployment has no symbol fields by domain rule. No `Type` field exists on `Deployment`.
