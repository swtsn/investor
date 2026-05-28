# ADR-008: Budget and allocation splits use truncation; last bucket absorbs remainder

## Status
Accepted

## Context

When splitting a total amount across buckets by `TargetPct` (in `PreviewBudget` / `ApplyBudget`) or across allocations by `TargetPct` (in `PreviewDeployment`), decimal multiplication can produce fractional cents. For example, splitting $100 three ways at 33.33% / 33.33% / 33.34% requires a decision about where the rounding residual goes.

Two options were considered:

1. Round each bucket independently to 2 decimal places — simple, but `Σ allocations` may differ from the input total by ±$0.01, causing `Σ Contributions ≠ BudgetEvent.TotalAmount`.
2. Truncate each bucket to 2 decimal places; assign the remainder (`total − Σ truncated`) to the last bucket — `Σ allocations` always equals the input total exactly.

## Decision

Option 2: truncate all buckets/allocations to 2 decimal places; the last bucket or allocation receives `total − Σ all others`. This applies to both `PreviewBudget`/`ApplyBudget` and `PreviewDeployment`.

## Consequences

`Σ Contributions` always equals `BudgetEvent.TotalAmount` exactly — pool balance arithmetic is clean. `PreviewBudget` and `ApplyBudget` use the same rounding rule, so the preview always matches what is written. The last bucket may receive a slightly different amount than a pure percentage would suggest (at most $0.01 difference), which is acceptable at the scale this app targets.
