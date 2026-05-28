# ADR-005: Use pure-Go SQLite driver (`modernc.org/sqlite`)

We chose `modernc.org/sqlite` over the more common `mattn/go-sqlite3`. Both wrap the same SQLite library, but `mattn` requires cgo — a C compiler at build time — which breaks `go install`, complicates cross-compilation, and causes friction on machines without a configured C toolchain. `modernc.org/sqlite` is a transpiled pure-Go port with no cgo dependency and equivalent feature coverage. For a local TUI binary, the build simplicity outweighs any marginal performance difference.
