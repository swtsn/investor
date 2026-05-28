# ADR-010: Single InvestorService gRPC transport; buf toolchain; string amounts

## Status
Accepted

## Context

Phase 4 introduces a client-server split: the server binary opens the SQLite DB and runs the service layer; the TUI binary is a network client. A transport protocol and proto schema structure are required.

trade-tracker-go uses gRPC with buf-generated code, multiple proto services split by domain (AccountService, PositionService, etc.), and string-encoded currency amounts.

For investor, three structural decisions were made:

**1. Single service vs. multiple**
investor has one tightly-coupled domain; all RPCs touch buckets and pools. Splitting into BudgetService, DeploymentService, PoolService mirrors the Go service layer but adds file overhead with no real benefit at 10 RPCs. Multiple services make sense when domains are independent (trade-tracker-go) or teams own them separately — neither applies here.

**2. Amount encoding**
The domain uses `shopspring/decimal` stored as SQLite TEXT. Options were `string`, `int64` cents, and `double`. `double` is wrong for money. `int64` cents adds a ×100 conversion at every boundary. `string` threads straight from SQLite TEXT → service layer decimal → proto → TUI display with zero conversion.

**3. Toolchain**
buf (remote plugins) is already in use in trade-tracker-go. No reason to diverge.

## Decision

- One `InvestorService` proto service with all RPCs in `proto/investor/v1/investor.proto`
- Generated code in `gen/investor/v1/`
- `buf.yaml` + `buf.gen.yaml` mirroring trade-tracker-go
- All monetary amounts encoded as `string` in proto messages

## Consequences

One proto file, one generated stub, one handler (`internal/grpc/investor_handler.go`), one client interface (`internal/tui/client/`). Amount values passed as strings end-to-end; the TUI parses user input via `decimal.NewFromString` and renders proto strings directly without conversion.
