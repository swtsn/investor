# ADR-001: Buckets are logical categories, not account-bound

## Status
Accepted

## Context

Each bucket conceptually maps to a brokerage or exchange account (e.g., investing → brokerage, crypto → exchange). However, this mapping may change — a single account might hold multiple logical buckets, or a bucket might eventually span accounts.

## Decision

Buckets are modeled as logical categories. No account identity in the data model at this stage.

## Consequences

Simpler initial model with no account plumbing. Future association of buckets to accounts will require a migration or an added join table. The UI has no account details to surface for MVP.
