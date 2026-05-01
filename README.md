[![GitHub release](https://img.shields.io/github/release/sgaunet/kubetrainer.svg)](https://github.com/sgaunet/kubetrainer/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/sgaunet/kubetrainer)](https://goreportcard.com/report/github.com/sgaunet/kubetrainer)
![GitHub Downloads](https://img.shields.io/github/downloads/sgaunet/kubetrainer/total)
![Coverage](https://raw.githubusercontent.com/wiki/sgaunet/kubetrainer/coverage-badge.svg)
[![linter](https://github.com/sgaunet/kubetrainer/actions/workflows/linter.yml/badge.svg)](https://github.com/sgaunet/kubetrainer/actions/workflows/linter.yml)
[![coverage](https://github.com/sgaunet/kubetrainer/actions/workflows/coverage.yml/badge.svg)](https://github.com/sgaunet/kubetrainer/actions/workflows/coverage.yml)
[![Snapshot Build](https://github.com/sgaunet/kubetrainer/actions/workflows/snapshot.yml/badge.svg)](https://github.com/sgaunet/kubetrainer/actions/workflows/snapshot.yml)
[![Release Build](https://github.com/sgaunet/kubetrainer/actions/workflows/release.yml/badge.svg)](https://github.com/sgaunet/kubetrainer/actions/workflows/release.yml)
[![Vulnerability Scan](https://github.com/sgaunet/kubetrainer/actions/workflows/vulnerability-scan.yml/badge.svg)](https://github.com/sgaunet/kubetrainer/actions/workflows/vulnerability-scan.yml)
![License](https://img.shields.io/github/license/sgaunet/kubetrainer.svg)

# KubeTrainer

KubeTrainer is a small Go application packaged as a Docker image, designed as a playground for learning core Kubernetes concepts. It exposes liveness/readiness probes, demonstrates graceful shutdown, and runs in two cooperating modes (web producer and stream consumer) so you can experiment with horizontal scaling and at-least-once message processing on Redis Streams.

This project is part of other projects:

* https://github.com/sgaunet/kubetrainer-docs : Documentation for kubetrainer
* https://github.com/sgaunet/helm-kubetrainer : Helm chart to deploy kubetrainer in a kubernetes cluster

## Features

KubeTrainer is built to make the following Kubernetes concepts visible and testable:

- **Liveness & readiness probes** — toggleable from the UI to simulate pod failures
- **Graceful shutdown** — handles `SIGTERM`/`SIGINT` with a 15s (web) / 120s (consumer) drain
- **Horizontal scaling** — multiple consumer replicas process Redis stream messages without loss (consumer groups, pending claim recovery)
- **CPU-intensive workloads** — consumer mode generates configurable random data and computes a SHA-256 hash, useful for HPA / resource-limit experiments
- **Health checks** — DB and Redis connectivity surfaced in the dashboard
- **Configuration patterns** — YAML file *or* environment variables (good fit for ConfigMaps)
- **Container best practices** — multi-stage build, `scratch`-based final image, non-root user

## Quick Start

### Run the prebuilt image

```bash
docker run --rm -p 3000:3000 ghcr.io/sgaunet/kubetrainer:latest
# Open http://localhost:3000
```

The web UI runs without Redis or PostgreSQL, but publishing messages and pending-message counts only work when Redis is configured.

### Full local stack (Redis + Postgres + producer + consumer)

```bash
cd deployment
docker compose up
```

This starts: Redis, PostgreSQL, the kubetrainer web producer, a consumer instance, and RedisInsight for browsing streams.

### Deploy on Kubernetes

Use the Helm chart: https://github.com/sgaunet/helm-kubetrainer

## Modes

KubeTrainer ships as a single binary with two modes:

| Mode | Flag | Purpose |
|------|------|---------|
| Web (default) | *(none)* | HTTP UI on `:3000`, probes, publishes messages to Redis |
| Consumer | `-consumer` | Reads from a Redis stream consumer group, simulates CPU work, no HTTP |

Run consumer mode locally:

```bash
go run ./cmd/*.go -consumer -f config.yml
```

## Configuration

Configuration can be provided via a YAML file (`-f config.yml`) or environment variables.

### YAML

```yaml
db:
  dbDsn: "postgres://user:pass@host:port/db?sslmode=disable"
redis:
  redisDsn: "redis://host:port/db"
  maxStreamLength: 1000
  redisStreamName: "stream_name"
  redisStreamGroup: "group_name"
producer:
  dataSizeBytes: 1073741824   # 1 GiB of random bytes per consumed message
```

### Environment variables

| Variable | Purpose |
|---|---|
| `DB_DBDSN` | PostgreSQL DSN (optional; UI degrades gracefully if unset) |
| `REDIS_REDISDSN` | Redis DSN |
| `REDIS_STREAMNAME` | Redis stream key |
| `REDIS_STREAMGROUP` | Consumer group name |
| `REDIS_MAXSTREAMLENGTH` | Cap on stream length (defaults to YAML value) |
| `PRODUCER_DATASIZEBYTES` | Bytes of random data hashed per consumed message |

See `pkg/config/config.go` for the full env-var mapping.

## HTTP Routes

| Method | Path | Purpose |
|---|---|---|
| GET | `/` | Status dashboard |
| GET | `/liveness` | Liveness probe (toggleable) |
| GET | `/readiness` | Readiness probe (toggleable) |
| GET | `/update-liveness` | Flip liveness state |
| GET | `/update-readiness` | Flip readiness state |
| POST | `/publish-time` | Publish one timestamp to Redis |
| POST | `/publish-time/{count}` | Publish *count* timestamps to Redis |

## Development

### Prerequisites

- Go 1.25+
- Docker (required for integration tests via testcontainers)
- [Task](https://taskfile.dev/) (optional, but used by all task targets below)

### Common tasks

```bash
task run          # generate templates + run the web server
task lint         # golangci-lint
task snapshot     # local multi-arch build via GoReleaser (no publish)
task release      # tagged release via GoReleaser
task doc          # godoc on http://localhost:6060
```

### Tests

```bash
go test ./... -v
go test -coverpkg=./... -coverprofile=profile.cov ./...
go tool cover -html=profile.cov
```

Database tests use `testcontainers-go` to spin up a real PostgreSQL container; Docker must be running.

### Templates

HTML is generated via [templ](https://templ.guide/). Regenerate after editing `internal/html/views/*.templ`:

```bash
go generate ./...
```

## License

[MIT](LICENSE) © Sylvain

## 🕐 Project Status: Low Priority

This project is not under active development. While the project remains functional and available for use, please be aware of the following:

### What this means:
- **Response times will be longer** - Issues and pull requests may take weeks or months to be reviewed
- **Updates will be infrequent** - New features and non-critical bug fixes will be rare
- **Support is limited** - Questions and discussions may not receive timely responses

### We still welcome:
- 🐛 **Bug reports** - Critical issues will eventually be addressed
- 🔧 **Pull requests** - Well-tested contributions are appreciated
- 💡 **Feature requests** - Ideas will be considered for future development cycles
- 📖 **Documentation improvements** - Always helpful for the community

### Before contributing:
1. **Check existing issues** - Your concern may already be documented
2. **Be patient** - Responses may take considerable time
3. **Be self-sufficient** - Be prepared to fork and maintain your own version if needed
4. **Keep it simple** - Small, focused changes are more likely to be merged

### Alternative options:
If you need active support or rapid development:
- Look for actively maintained alternatives
- Reach out to discuss taking over maintenance

We appreciate your understanding and patience. This project remains important to us, but current priorities limit our ability to provide regular updates and support.
