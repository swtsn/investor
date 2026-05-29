# ADR-014: Plan view — split-panel card layout, iterative, read-only

## Status
Accepted

## Context

A new Plan view is needed to answer "if I deploy $X, where does it go?" at both the bucket and allocation level. The view is purely informational — no data is written.

Three interaction models were considered:

1. **Drill-down** — enter amount → bucket list → select bucket → see its allocation splits. Mirrors the Deploy flow.
2. **All-at-once table** — hierarchical table with bucket rows and indented allocation sub-rows.
3. **Split-panel with cards** — left panel holds the amount input; right panel shows one card per bucket, arranged horizontally, each card showing pool balance, planned amount, and allocation splits.

## Decision

Split-panel card layout (option 3).

- Left panel: amount text input. Cards populate on Enter (consistent with Budget/Deploy preview pattern). Input remains accessible after preview so the user can iterate with different amounts.
- Right panel: one lipgloss-bordered card per bucket, arranged side by side. Each bucket gets a distinct accent color. Card content: bucket name, target%, current pool balance, planned amount for this bucket, and (for diversified buckets) each allocation with its target% and planned dollar amount. Flat buckets show amount only — no allocations.
- On load: cards render immediately with pool balance data (from `GetDashboard`); planned amounts show "—" until an amount is entered.
- Navigation key: `p`. Added to the help bar as `[p]lan`.

## Consequences

- `PlanView` fetches `GetDashboard` on load for pool balances, then `PreviewBudget` + `PreviewDeployment` per diversified bucket on each Enter.
- A fixed color palette cycles across bucket cards (lipgloss foreground/border colors).
- If cards are too narrow (many buckets on a small terminal), the layout degrades — acceptable for now; wrapping or scrolling can be added later.
- Future: a "make it so" action from this view could execute the plan (flat buckets as amount-only deployments; diversified buckets routed into an allocation entry flow). Deferred until the read-only view is validated in use.
