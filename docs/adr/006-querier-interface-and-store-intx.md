# ADR-006: `querier` interface + `Store.InTx` for transaction composability

Repository implementations accept a `querier` interface (satisfied by both `*sql.DB` and `*sql.Tx`) rather than holding `*sql.DB` directly. This lets the same repo code run inside or outside a transaction without branching.

Cross-repo atomicity (e.g. `ApplyBudget` writing a `BudgetEvent` + multiple `Contribution`s) is provided by `Store.InTx(ctx, func(*Store) error)`, which constructs a tx-scoped `*Store` with all repos backed by the same `*sql.Tx` and passes it to the callback. Transaction boundaries are the caller's responsibility — repos do not start their own transactions.

The alternative — each repo managing its own transactions internally — breaks down the moment two repos need to participate in the same atomic write. We ruled it out rather than discover the limitation mid-Phase 3.
