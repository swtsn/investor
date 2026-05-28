# ADR-007: `ApplyBudget` distinguishes month-opening from top-up calls

## Status
Accepted

## Context

A user may call `ApplyBudget` more than once in a calendar month — for example, an initial budget at the start of the month and a top-up when additional funds become available later. Slush (undeployed carry-over) must only be auto-generated once per month: at the moment the new month is "opened". If slush were generated on every `ApplyBudget` call, the second call would see the first call's undeployed budget contributions as carry-over and write them as slush, doubling the pool balance.

## Decision

`ApplyBudget` checks whether any `BudgetEvent` already exists for the given calendar month (via `BudgetEventRepository.ListByMonth`) before opening a transaction:

- **Month-opening call** (no existing `BudgetEvent` for the month): compute pool balance per bucket (`Σ all-time contributions − Σ all-time deployments`), write a slush `Contribution` per bucket if balance > 0, then write budget `Contributions`. Slush `Contributions` carry the new `BudgetEventID`.
- **Top-up call** (a `BudgetEvent` already exists for the month): write budget `Contributions` only. No slush logic runs.

The alternative — blocking multiple `BudgetEvent`s per month — was rejected because top-ups are a real use case and the domain has no concept of a "closed" month.

## Consequences

Multiple budget events per month are supported. Slush is never double-counted. The month-open/top-up distinction is implicit (derived from whether a `BudgetEvent` exists for the month) rather than an explicit field on `BudgetEvent`. Retroactive edits to a prior month's data are not self-correcting — slush is written once and not recalculated.
