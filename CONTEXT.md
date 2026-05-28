# investor — Domain Context

## Purpose

A TUI application for planning and tracking investment diversification and allocation across logical investment buckets. Helps answer: how much goes where, how much is left to deploy, and how am I tracking against my diversification targets.

## Domain Glossary

**Bucket** — A logical investment category (e.g., investing, options, crypto). Each bucket has a target percentage of the monthly budget. Two types: `flat` (options — no per-symbol diversification) and `diversified` (symbol-level allocation targets apply).

**Allocation** — A named target percentage within a diversified bucket (e.g., "metals", "large-cap tech", or a single ticker when it maps 1:1). Allocations for a bucket should sum to 100%. The symbol deployed against an Allocation is recorded on the Deployment, not the Allocation itself — a single Allocation may be filled by multiple symbols in the same period.

**Pool** — The running balance of available-to-deploy funds within a bucket. `pool balance = Σ contributions − Σ deployments`.

**Contribution** — Money entering a bucket's pool, tagged with an origination: `budget` (from a monthly budget event), `reinvestment` (realized gains/losses re-entering the market, entered manually by the user), or `slush` (undeployed carry-over from the prior month, auto-generated when a new budget event is applied). Budget and slush contributions carry a BudgetEventID; reinvestment contributions do not.

**BudgetEvent** — A monthly action where the user enters a total investment amount. The app splits it across buckets by target percentages and creates Contribution records.

**Deployment** — Money leaving a bucket's pool via an actual investment (fill). Always records an amount. The user may record a full fill (symbol + shares + price per share) or an amount-only deployment for experimental or untracked trades. Flat bucket deployments are always amount-only (no symbol). Shares and price per share are always provided together; if both are present, amount is derived as shares × price per share. Funded by one or more Contributions via DeploymentSource records; the sum of DeploymentSource amounts must equal the Deployment amount.

**DeploymentSource** — A join between a Deployment and a Contribution, recording how much of a given Deployment was drawn from a specific Contribution. Enables a single Deployment to draw from multiple funding sources (e.g., part budget, part slush).

**Slush** — Undeployed funds carried over from a prior period. Modeled as an explicit `Contribution` with `origination=slush`. Auto-generated per bucket on the *first* `BudgetEvent` of a new month: `slush_amount = Σ all-time contributions − Σ all-time deployments` (the running pool balance at that moment). Subsequent `BudgetEvent`s in the same month (top-ups) do not generate slush. Only written if `slush_amount > 0`.

**Month** — A navigable view of all events within a calendar month. Derived from event timestamps; not a stored entity.

**Dry Powder Target** — (future) A per-bucket percentage of the pool to intentionally leave uninvested for opportunistic trades. Not yet modeled.

## Core Rules

1. Options bucket is flat — no symbol allocations; deployments are dollar amounts only.
2. Diversified buckets split a deployment amount across symbols at decision time using current allocation percentages.
3. Month is always a rollup; there is no concept of opening or closing a month.
4. All prices are entered manually at fill time — no live market data.

## Out of Scope

- Trade-level P&L, position history, Greeks → trade-tracker-go
- Live price feeds
- Tax lot tracking
