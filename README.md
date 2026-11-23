# nimbus-rbac-stage

A production-ready Go API template for microservices following clean architecture and domain-driven design principles.

## Features

### 🏗️ Architecture
- **Clean Architecture** with domain-driven design
- **Microservice-ready** with service-to-service communication
- **Health checks** with database connectivity monitoring
- **Caching** for health checks to reduce database load

### 🔐 Security
- **JWT authentication** for service-to-service communication
- **Rate limiting** middleware with a token bucket algorithm
- **Request ID tracking** for distributed tracing
- **Input validation** and sanitization

### 🚀 Performance & Reliability
- **Circuit breaker** pattern for resilient outbound calls
- **Connection pooling** for database connections
- **Prometheus metrics** collection
- **Structured logging** with slog

### 🐳 Deployment
- **Docker** multi-stage builds
- **GitHub Actions** CI/CD pipeline
- **Testcontainers** for integration testing

## Quick Start

### Prerequisites
- The Go version in `go.mod` (1.27.4)
- Docker Engine with BuildKit and the Docker Compose plugin
- PostgreSQL with the `uuid-ossp` extension available

### Local Development

```bash
# Clone the repository
git clone https://github.com/acheevo/project--000001a2-09d2-7270-b693-465d2e4f237a.git
cd nimbus-rbac-stage

# Set up development environment
make dev-setup

# Run with Docker Compose (includes PostgreSQL)
make docker-run

# Or start just PostgreSQL and run the Go process locally
docker compose up -d --wait postgres
make run
```

### Available Commands

```bash
make help           # Show all available commands
make build          # Build the application
make test-unit      # Run unit tests without Docker
make test           # Run unit and integration tests
make lint           # Run golangci-lint with integration files enabled
make ci             # Run full CI pipeline locally
make docker-build   # Build Docker image
make docker-run     # Run with Docker Compose
```

## Configuration

The application uses environment variables for configuration:

```env
# Server
HTTP_ADDR=:8080
LOG_LEVEL=info
ENVIRONMENT=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=api_template
DB_SSL_MODE=disable
DB_MAX_IDLE_CONNS=5
DB_MAX_OPEN_CONNS=25
DB_CONN_MAX_LIFETIME=300s

# Authentication
JWT_SECRET=development-jwt-secret
JWT_EXPIRATION=24h

# Service Identity (for service-to-service calls)
SERVICE_NAME=nimbus-rbac-stage
SERVICE_VERSION=1.0.0
```

See `.env.example` for the full configuration. Supply an explicit, randomly generated
JWT secret of at least 32 bytes in production. Use a dedicated database role and
enable certificate verification with `DB_SSL_MODE=verify-full` for a remote
production database. The startup role needs access to an existing `uuid-ossp`
extension or permission to create it.

## API Endpoints

### Health & Monitoring
- `GET /health` - Detailed health check (cached)
- `GET /ready` - Readiness probe (always fresh)
- `GET /live` - Liveness probe
- `GET /metrics` - Prometheus metrics

### API Routes (JWT Protected)
- `GET /api/v1/info` - Service information with system metrics

## Architecture Overview

```
cmd/
├── api/                    # Application entry point
internal/
├── auth/service/           # JWT service
├── client/                 # HTTP client with circuit breaker
├── health/                 # Health check domain
├── info/                   # Service info domain
├── middleware/             # HTTP middlewares
├── http/                   # HTTP server setup
└── shared/                 # Shared utilities
    ├── config/             # Configuration management
    ├── database/           # Database connection
    └── logger/             # Structured logging
tests/
└── integration/            # Integration tests with testcontainers
```

## Testing

The template includes comprehensive testing:

```bash
# Unit tests
make test-unit

# Integration tests (uses testcontainers)
make test-integration

# All tests
make test
```

Integration tests use the `integration` build tag and start temporary PostgreSQL
containers using Testcontainers. Start the Docker daemon before running
`make test-integration` or `make ci`.

Install the same lint version as CI with:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
```

## Deployment

### Docker

```bash
# Build image
docker build -t nimbus-rbac-stage .

# Run the image with an existing PostgreSQL instance
docker run --rm -p 8080:8080 --env-file /path/to/api.env nimbus-rbac-stage
```

Use the variables above in `api.env`. Set `DB_HOST` to a host reachable from
the container. The image runs as a non-root user and uses `/ready` as its
health check. In an orchestrator, use `/live` for liveness and `/ready` for
readiness.

### Upgrade an existing local PostgreSQL 15 volume

The default Compose database is PostgreSQL 18. Existing projects that used the
old `postgres_data` volume retain their data in that volume. PostgreSQL 18 uses
the separate `postgres_18_data` volume and a new upstream data layout.

Before switching versions, stop writes and dump the old database while running
the previous Compose configuration:

```bash
docker compose stop api
docker compose exec -T postgres pg_dump -U postgres -d api_template -Fc > api_template.dump
docker compose down
```

After updating, start only PostgreSQL, restore the dump, then start the API:

```bash
docker compose up -d --wait postgres
docker compose exec -T postgres pg_restore --exit-on-error -U postgres -d api_template < api_template.dump
docker compose up -d --build --wait api
```

Keep the dump and the original volume until you have verified the restored data.

## Microservice Communication

Services communicate using JWT tokens with shared secrets:

```go
// Example service-to-service call
client := client.NewHTTPClient(client.HTTPClientConfig{
    ServiceName: "my-service",
    ServiceID:   "service-001", 
    Version:     "1.0.0",
    Environment: "production",
    JWTService:  jwtService,
    Logger:      logger,
})

resp, err := client.Get(ctx, "http://other-service/api/v1/data", nil)
```

## Monitoring & Observability

### Metrics
- HTTP request metrics (duration, status codes)
- Circuit breaker state and counts
- Database connection pool stats
- Custom business metrics

### Health Checks
- **Health** (`/health`): Cached checks for performance
- **Readiness** (`/ready`): Fresh checks for K8s probes
- **Liveness** (`/live`): Simple alive check

### Logging
Structured JSON logging with:
- Request ID correlation
- HTTP request/response logging
- Error tracking with context
- Performance metrics

## Development Guidelines

### Adding New Domains
1. Create domain in `internal/newdomain/`
2. Follow clean architecture layers:
   - `domain/` - entities and interfaces
   - `service/` - business logic
   - `transport/` - HTTP handlers
3. Register routes in `internal/http/server.go`
4. Add tests in `tests/integration/`

### Code Standards
- All code must pass `golangci-lint`
- Follow Go naming conventions
- Use structured logging
- Handle errors explicitly

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes following the code standards
4. Run `make ci` to ensure all checks pass
5. Submit a pull request

## License

This template is provided as-is for building microservices. Customize as needed for your organization.

## Typed HTTP workflow

The existing Gin router owns middleware and probes. Huma registers the typed
`GET /api/v1/info` operation in the JWT-protected group. Start new versioned
routes with a stable operation ID, typed input and output, and explicit
authorization. Use Huma input tags such as `query`, `path`, or `minLength`
when an operation needs validation.

Run `make api-contract` to regenerate [OpenAPI](docs/openapi.json) without a
database. With the app running in development, read `GET /openapi.json` on port
8080. CI compares the generated document with the committed one. Run a focused
HTTP test alongside the contract check so middleware and response codes are
covered as well as the schema.

## Managed AI

In Nimbus, choose **AIaaS** under **Create project → Environment → Managed Services**.
The first deployment receives `AIAAS_URL` and `AIAAS_API_TOKEN` as server-side
runtime variables. `internal/managedai.FromEnvironment()` returns a Go client.
Use `client.Run(ctx, managedai.Prompt{Provider: "claude", Prompt: "Summarize this text"})`
from a trusted backend workflow. The starter also mounts `POST /api/v1/ai/run`
behind its service JWT middleware when AIaaS and a strong `JWT_SECRET` are
configured. Set a randomly generated signing key through Nimbus Runtime Secrets. Send a `provider`,
`prompt`, and optional `model` as JSON. The response contains `provider` and `result`.
