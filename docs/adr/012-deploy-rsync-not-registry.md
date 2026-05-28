# ADR-012: Deploy via rsync + remote Docker build, not a registry

The server binary is cross-compiled locally (`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`), then rsynced with the Dockerfile to the target host, where `docker build` and `docker compose up` run over SSH. No image registry is involved.

The alternative — push to a registry (Docker Hub, GHCR, or self-hosted) then pull on the server — requires registry credentials on both the build machine and the server, a registry account or service, and a push step that creates a public or semi-public artifact. For a single personal server reachable over SSH, that overhead has no payoff. The rsync approach needs only SSH access and keeps the entire flow local: build, ship, run.

The Dockerfile deliberately has no build stage. The binary is compiled by the Makefile (`make release-server`), not inside Docker, so the image stays minimal (distroless base + binary) and the build cache lives on the developer's machine where it is warm.
