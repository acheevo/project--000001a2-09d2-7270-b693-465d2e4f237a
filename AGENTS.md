# API template agent guide

This repository is a Go service starter. Projects generated from it keep the same layout and commands. Treat the HTTP API, environment variables, container entry point and release workflow as part of the template contract.

## Take ownership of the task

1. Read `git status --short`, the relevant code, and the nearest tests. Keep pre-existing edits intact. Trace behavior from `cmd/api/main.go` through `internal/http/server.go` to the handler, service and database before changing a contract.
2. Identify the expected outcomes and pick the smallest check that can prove each one. Reproduce a bug at the normal entry point before fixing it when practical.
3. Make a focused change using the existing patterns below. Update callers, samples and tests that describe the changed behavior. For an independent task, use an isolated checkout and coordinate overlapping edits.
4. Run the focused check, then the applicable repository checks. Investigate the first meaningful failure, fix it, and rerun. If one check is blocked, keep running independent checks and save the exact failure for the handoff.
5. Review `git diff --check` and the full diff. Report behavior changes, any migration, and the exact commands and results. Include the PR and CI results when working on GitHub.

## Where changes go

| Area | Entry points and expectations |
| --- | --- |
| Startup and shutdown | `cmd/api/main.go` wires configuration, the database pool, services, and graceful termination. |
| Routing and request handling | `internal/http/server.go` owns middleware order and route registration. See `internal/health/transport/` and `internal/info/transport/` for handler conventions. |
| Domain behavior | Keep business logic in feature packages under `internal/<feature>/service/` and payloads and interfaces under `internal/<feature>/domain/`. Keep transport and persistence details out of the domain. |
| Auth and outbound requests | `internal/auth/service/jwt.go`, `internal/middleware/jwt.go`, and `internal/client/` cover bearer validation, service claims, deadlines, and circuit-breaker behavior. |
| Configuration and database | `internal/shared/config/`, `internal/shared/database/`, and `.env.example` define environment inputs, safe defaults, PostgreSQL configuration and pooling. |
| Delivery | `Dockerfile`, `docker-compose.yml`, and `.github/workflows/` define build, runtime, CI and the tag-driven Docker Hub release. |

Keep `GET /live` for process liveness, `GET /ready` for fresh database readiness, and `GET /health` for detailed status. API routes under `/api/v1` go behind JWT middleware. For a route change, exercise expected response, invalid input and unauthorized access as applicable. Propagate request context to dependency calls, preserve error causes, close response bodies you own and avoid logging credentials or tokens. Keep Prometheus label values bounded.

For a DB change, test against a real PostgreSQL container, include a migration path when existing storage is affected and exercise failure and recovery. For a config change, update `.env.example` and validate both configured and default startup. Use disposable credentials and volumes for local validation.

## Verify

| Scope | Commands |
| --- | --- |
| Go toolchain and dependencies | `go version`, `go mod download`, `go mod tidy -diff`, `go mod verify` |
| Focused tests | `go test -race ./internal/client/...` or `go test -race -tags=integration ./tests/integration -run TestHealthIntegrationSuite` |
| Default tests without Docker | `make test-unit` |
| PostgreSQL integration tests | `make test-integration` |
| Local CI | `make ci` with a running Docker engine and the golangci-lint version in `.github/workflows/ci.yml` |
| Workflows and deployment | `actionlint`, `docker compose config --quiet`, `docker build -t api-template .`, then the Compose smoke check below |
| Dependency changes | `go list -m -u all`, `govulncheck -test -tags=integration ./...`, then `make ci` |

For a runtime smoke check, choose a unique Compose project and available host ports. Use the same project and environment on subsequent commands.

```sh
export API_PORT=18080 POSTGRES_PORT=15432
docker compose -p api-smoke up --build --wait
curl -fsS http://127.0.0.1:18080/live
curl -fsS http://127.0.0.1:18080/ready
curl -fsS http://127.0.0.1:18080/health
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:18080/api/v1/info
docker compose -p api-smoke down -v
```

Expect HTTP 200 for the three probes and HTTP 401 for the unauthenticated info request. Remove volumes only from your disposable Compose project. Shared environments or release tags affect deployed systems.

## Recover common failures

- If the compiler, linter or stdlib versions disagree, check `go version`, `go env GOROOT GOTOOLCHAIN`, and `golangci-lint version`. Align the toolchain with `go.mod` and the Docker builder, clear stale Go environment overrides in the current shell, and use a linter built with a compatible Go version.
- If integration setup cannot find Docker, check `docker info` and the selected context or socket. If a test container starts but fails readiness, inspect its logs and image tag before retrying.
- If Compose starts the API but the probe fails, use `docker compose ps` and `docker compose logs api postgres`. Check port mappings, DB credentials, extension permissions and the distinction between DB readiness and process liveness.
- For a failing remote check, use `gh pr checks` and inspect the failed steps with `gh run view --log-failed`. Reproduce the same target locally and let checks finish on the new commit.
- For a toolchain bump, update `go.mod` and the Docker builder together. Check major-version changes in official release notes. Keep both Testcontainers modules and both PostgreSQL image references aligned. Keep the workflow's linter version and the README install command aligned.

## Change a typed HTTP contract

Start from the Huma registration in `internal/info/transport/http.go` and the surrounding Gin middleware in `internal/http/server.go`. Use request contexts for service calls and preserve the HTTP status, error shape, and authorization policy. Run `make api-contract` and inspect the resulting contract diff. Verify the route through the registered HTTP stack.
