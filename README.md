# Business platform (microservices)

Go services that expose gRPC APIs and use Kafka for some async flows.

Services in this repo:

`user-service` handles auth and users. It uses PostgreSQL.

`inventory-service` handles products. It uses PostgreSQL, exposes HTTP and gRPC, and can consume Kafka messages.

`order-service` handles orders. It uses PostgreSQL, exposes HTTP and gRPC, publishes order events to Kafka, and calls inventory over gRPC.

`api-gateway` is the HTTP entry point; see its config and app package for how it reaches other services.

Persistence: user, order, and inventory data use PostgreSQL (separate logical databases: `user_service`, `order_service`, `inventory_service` when using the provided compose init script).

Infra: `docker-compose.yml` can run Postgres, Kafka, Zookeeper, and Redis. Redis is optional for the current Go code. Kafka matters if you use the order producer and inventory consumer. You can run databases on the host instead of Docker if you prefer.

Prerequisites: Go (version in each `go.mod`), Task for codegen tasks, protoc and Go gRPC plugins if you edit protos. You need Postgres for all three data services, and Kafka only if you exercise those code paths.

Docker (optional):

```bash
docker compose up -d
```

Postgres in Compose is exposed on port 5432 with default database `order_service` and user `postgres` unless you change the file. On a fresh data volume, `deploy/postgres-init` also creates `user_service` and `inventory_service` databases. Redis in Compose expects `REDIS_PASSWORD` in an `.env` beside `docker-compose.yml` or in the environment.

Configuration: each service reads env vars via caarlos0/env and loads `local.env` in its folder. Copy from the sample `.env` to `local.env` and adjust. Do not commit secrets.

Typical vars: `GRPC_PORT`, `GRPC_TIMEOUT`, `HTTP_PORT` (required where HTTP is enabled), optional `VERSION`. User-service, inventory-service, and order-service each need `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB` (use the matching database name per service), `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_SSL_MODE`. Order-service also needs `BROKERS` for Kafka, `INVENTORY_SERVICE_HOST`, and `INVENTORY_SERVICE_PORT`. User-service needs SMTP settings for mail. Inventory needs `BROKERS` for the consumer.

Protobuf: from a service directory that has a Taskfile, run `task generate`. Output is under `protos/gen/golang/`.

Run a service:

```bash
cd <service-directory>
go run ./cmd/main.go
```

Start inventory before order-service if orders call inventory. Have Postgres running with the correct `POSTGRES_DB` per service.

Tests example:

```bash
cd user-service
go test ./...
```

Note: the folder name uses Plantform. Modules use `github.com/19parwiz/...` paths. Use `git push` to publish commits, not `go push`.
