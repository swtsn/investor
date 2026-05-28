# ADR-002: Month is a derived rollup, not a stored entity

## Status
Accepted

## Context

The user navigates the app by calendar month to see allocation summaries for a given period. Two options: (a) store Month as a first-class entity that events belong to, or (b) derive monthly views by filtering events by timestamp.

## Decision

Month is derived. All events carry timestamps; monthly views are aggregations over those timestamps. There is no concept of opening or closing a month.

## Consequences

Simpler write path — events are just events. Historical corrections are easy (adjust an event's timestamp or add a backdated event). Monthly boundaries can be queried ad hoc. The read/aggregation path is slightly more complex but appropriate for the data volumes involved.
