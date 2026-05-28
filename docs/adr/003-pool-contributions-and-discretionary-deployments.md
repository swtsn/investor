# ADR-003: Flat contribution pool with origination tags; slush deployments are discretionary

## Status
Accepted

## Context

Money enters a bucket from multiple sources: monthly budget allocation and reinvestment of realized gains. Undeployed money accumulates as slush over time. The user may want to deploy slush to experimental trades that don't follow normal diversification targets — slush money is treated differently from new allocated money.

## Decision

- All money entering a pool is recorded as a **Contribution** with an origination tag: `budget`, `reinvestment`, or `slush`.
- Slush is an explicit `Contribution` with `origination=slush`, auto-generated per bucket when a new `BudgetEvent` is applied. It is not derived at read time.
- When `ApplyBudget` runs for a new month, it computes the remaining pool balance per bucket from the prior month and writes a slush Contribution (if non-zero) before writing the new budget Contributions: `slush_this_month = prev_slush + prev_month_remaining_budget`.
- **Deployments** carry a type: `allocated` (respects symbol diversification rules, records shares + price per share) or `discretionary` (free-form symbol choice, used for slush/experimental trades, bypasses diversification calculations).
- The UI surfaces origination breakdown — new budget, reinvestment, and carry-over slush — as a simple group-by over origination types within the current month.

## Consequences

Slush is queryable and auditable without arithmetic over unbounded history. `PoolBreakdown` is a plain group-by with no special derivation logic. The trade-off is that slush must be written explicitly on each `ApplyBudget` call — it is not self-correcting if a prior month's data is edited retroactively.
