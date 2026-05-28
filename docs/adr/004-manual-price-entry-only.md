# ADR-004: Manual price entry only — no live market data

## Status
Accepted

## Context

Many investment apps integrate live price feeds to track current portfolio value. This app focuses on allocation planning and deployment tracking rather than real-time valuation.

## Decision

All prices are entered manually at fill time (shares + price per share). No external market data API is integrated.

## Consequences

No API keys, rate limits, or network dependency. Historical portfolio valuation requires the user to have entered fills. Live market data integration can be added later as an enhancement without architectural changes — the fill model already captures what a price feed would need to update.
