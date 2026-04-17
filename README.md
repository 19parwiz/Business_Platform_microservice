# Business platform (microservices)

Go services that expose gRPC APIs and use Kafka for some async flows.

Services in this repo:

`user-service` handles auth and users. It uses PostgreSQL.

`inventory-service` handles products. It uses PostgreSQL, exposes HTTP and gRPC, and can consume Kafka messages.

`order-service` handles orders. It uses PostgreSQL, exposes HTTP and gRPC, publishes order events to Kafka, and calls inventory over gRPC.

`api-gateway` is the HTTP entry point; see its config and app package for how it reaches other services.

Persistence: user, order, and inventory data share **one** PostgreSQL database named **`flower_shop`** (Flower Shop). Each service uses its own tables; ID counters use per-service tables (`user_auto_inc_ids`, `inventory_auto_inc_ids`, `order_auto_inc_ids`) so nothing collides.

### Database (read this before testing)

**Name:** `flower_shop` is a clear, conventional identifier (lowercase + underscore). Display branding can stay “Flower Shop”; in Postgres, mixed case like `Flower_Shop` needs quoted identifiers and is awkward in URLs, so this repo standardizes on `flower_shop`.

**One database, three services:** set the same `POSTGRES_DB=flower_shop` in **user-service**, **inventory-service**, and **order-service** `local.env` files (create `local.env` in each service directory; it is gitignored).

**With Docker Postgres** (`docker compose up -d postgres …`): `POSTGRES_DB` in `docker-compose.yml` is `flower_shop`, so the first volume init creates that database. Scripts under `deploy/postgres-init/` are only comments now (no extra `CREATE DATABASE`).

**If you still have an old volume** from the previous three-database layout, remove it once so Postgres starts clean: `docker compose down -v` (this **deletes all DB data** in that volume), then `docker compose up -d postgres` again.

**Local Postgres (no Docker):** create a single database `flower_shop`, then use the same `POSTGRES_DB` in all three services’ `local.env`.

Tables are created automatically on service startup (`CREATE TABLE IF NOT EXISTS` in each service’s `pkg/postgres`).

Infra: `docker-compose.yml` can run Postgres, Kafka, Zookeeper, and Redis. Redis is optional for the current Go code. Kafka matters if you use the order producer and inventory consumer. You can run databases on the host instead of Docker if you prefer.

Prerequisites: Go (version in each `go.mod`), Task for codegen tasks, protoc and Go gRPC plugins if you edit protos. You need Postgres for all three data services, and Kafka only if you exercise those code paths.

**Go modules:** if `go test` or `go run` fails with `GOPROXY list is not the empty string, but contains no entries`, reset your proxy once:

```bash
go env -w GOPROXY=https://proxy.golang.org,direct
go env -w GOSUMDB=sum.golang.org
```

Or run all module tests from the repo root with `scripts/go-test-all.ps1` (Windows) or `scripts/go-test-all.sh` (Linux/macOS), which set sane defaults for that session.

Docker (optional):

```bash
docker compose up -d
```

Postgres in Compose is exposed on port **5432** with user **postgres** / password **postgres** unless you change the file. Redis uses **`REDIS_PASSWORD`** from a root `.env` if present; otherwise it defaults to **`postgres`**.

Configuration: each service reads env vars via caarlos0/env and loads **`local.env`** in its service folder (you create this file; do not commit it). See the “Typical vars” line below for required keys.

Typical vars: `GRPC_PORT`, `GRPC_TIMEOUT`, `HTTP_PORT` (required where HTTP is enabled), optional `VERSION`. User-service, inventory-service, and order-service each need `POSTGRES_HOST`, `POSTGRES_PORT`, **`POSTGRES_DB=flower_shop`** (same value for all three), `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_SSL_MODE`. Order-service also needs `BROKERS` for Kafka, `INVENTORY_SERVICE_HOST`, and `INVENTORY_SERVICE_PORT`. User-service needs SMTP settings for mail. Inventory needs `BROKERS` for the consumer.

Protobuf: from a service directory that has a Taskfile, run `task generate`. Output is under `protos/gen/golang/`.

Run a service:

```bash
cd <service-directory>
go run ./cmd/main.go
```

Start inventory before order-service if orders call inventory. Have Postgres running with database **`flower_shop`** and the same `POSTGRES_DB` in every data service `local.env`.

Tests example:

```bash
cd user-service
go test ./...
```

**Local apps + Docker infra:** keep `POSTGRES_HOST=localhost` and `BROKERS=localhost:9092` in each `local.env` while Postgres/Kafka run in Docker with published ports. **Later, if you run Go services inside Docker too**, switch hostnames to the compose service names (`postgres`, `kafka`) and the internal Kafka port, not `localhost`.

Service order when starting manually: **inventory-service → user-service → order-service → api-gateway** (order depends on inventory gRPC; gateway depends on all three gRPC ports).

Note: the folder name uses Plantform. Modules use `github.com/19parwiz/...` paths. Use `git push` to publish commits, not `go push`.
