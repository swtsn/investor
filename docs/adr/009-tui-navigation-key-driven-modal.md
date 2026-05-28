# ADR-009: TUI navigation is key-driven modal from a persistent Dashboard

## Status
Accepted

## Context

The TUI has five views: Dashboard, Month navigator, Budget entry, Deploy entry, and Reinvest entry. The reference implementation (trade-tracker-go) uses a numbered-tab model (keys 1–6, tab/shift+tab to cycle) where all views are peers.

Two models were considered:

1. **Numbered tabs** — same pattern as trade-tracker-go. All views visible in a header tab bar; number keys and tab/shift+tab switch between them.
2. **Key-driven modal from Dashboard** — Dashboard is the persistent home screen. Budget, Deploy, and Reinvest are short-lived action views entered by a single key (`b`, `d`, `r`) and exited with `esc`. Month navigator is entered with `m`. No tab bar.

## Decision

Key-driven modal (option 2). Dashboard is the natural home screen — pool balances are the information you always want to see. Budget, Deploy, and Reinvest are *actions* (enter data, confirm, return), not persistent browsing views. The tab model made sense in trade-tracker-go because all views are independent browsing surfaces; here three of five views are transient forms.

## Consequences

`app.go` holds a `mode` field (dashboard / month / budget / deploy / reinvest) instead of a `ViewID` iota. `esc` from any action view returns to Dashboard. Global key handling only fires when in Dashboard mode. The help bar always shows `[b]udget  [d]eploy  [r]einvest  [m]onth  [q]uit`.
