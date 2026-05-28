# investor
Another vibey financial app to track diversification and allocation

## Prerequisites

- Go 1.22+
- [`golangci-lint`](https://golangci-lint.run/welcome/install/) — required by `make lint` / `make build`

```bash
brew install golangci-lint   # macOS
# or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Commands

```bash
make build          # fmt → vet → lint → test → build both binaries into bin/
make build-tui      # build TUI binary only (no quality gates)
make build-server   # build server binary only (no quality gates)
make test           # fmt → vet → lint → go test
make release-server # cross-compile server for linux/amd64 → bin/investor-linux
make deploy         # release-server + rsync to $(HOST) + docker compose up
make clean          # remove bin/
```

## Configuration

Copy `.env.example` to `.env` and set `HOST` to your SSH target before running `make deploy`.

The server reads `INVESTOR_DB` (default: `~/.investor/investor.db`) and `INVESTOR_ADDR` (default: `localhost:50051`). Both can be set as env vars or CLI flags.
