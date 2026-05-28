# ADR-011: Deploy flow — one deployment per invocation; shares×price drives amount; manual source picker

## Status
Accepted

## Context

The deploy view is the most complex in the TUI. Three non-obvious design decisions were made during Phase 4 planning.

**1. One deployment per invocation vs. all allocations at once**

For a diversified bucket with N allocations, `RecordDeployment` writes one `Deployment` record with one `AllocationID`. Deploying across all allocations requires N calls. The question was whether the deploy UI should loop through all allocations in a single flow (writing N deployments on confirm) or record one allocation per flow invocation.

All-at-once would require a loop-within-a-form in bubbletea and either N sequential RPC calls or a new batch RPC. It also forces the user to fill out symbol+shares+price for every allocation in one sitting.

One-per-invocation keeps the state machine linear and each deployment independently confirmable. The `PreviewDeployment` split table is shown at `AllocationPick` as informational context (how much should go to each allocation), not as a form to fill out all at once.

**2. Shares×price drives amount**

The domain requires `amount == shares × price` when both are provided, and `amount` is always present. Two models were considered: fix amount first (entered at the start, shares+price validated against it) vs. derive amount from shares×price (entered before source picking, amount computed).

Fixing amount first creates a conflict when the user enters shares+price that produce a different total than the pre-entered amount. Deriving amount from shares+price eliminates the conflict: the source picker always runs after amount is resolved, whether it came from manual entry (amount-only path) or from shares×price (full fill path).

**3. Manual cursor picker for deployment sources**

`ListDeployableSources(bucketID)` returns contributions with `remaining > 0`. The user picks which contributions fund the deployment and enters an amount per source (must sum to deployment amount). An auto-allocate alternative (draw from oldest first, no user input) was considered for Phase 4 but deferred — users wanted explicit control over which contributions are drawn.

## Decision

1. One `Deployment` per deploy flow invocation. `AllocationPick` shows `PreviewDeployment` splits as read-only context.
2. For full fills, `SymbolEntry → SharesEntry → PriceEntry` precedes source picking; amount = shares × price. For amount-only, `AmountEntry` precedes source picking. Flat buckets are always amount-only.
3. Manual cursor picker: arrow keys to navigate contributions, `enter` to open inline amount input per row, running total shown. Advance when Σ sources == deployment amount.

## Consequences

Deploy state machine per path:

- Flat: `BucketSelect → AmountEntry → SourcePick → Confirm`
- Diversified, amount-only: `BucketSelect → FillType → AmountEntry → AllocationPick → SourcePick → Confirm`
- Diversified, full fill: `BucketSelect → FillType → SymbolEntry → SharesEntry → PriceEntry → AllocationPick → SourcePick → Confirm`

`esc` at any step goes back one step. Date is editable on the Confirm screen (defaults to today). Deploy state machine may be revisited after first use.
