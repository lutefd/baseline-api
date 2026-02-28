# baseline-api

Go monolith for raw session ingestion, sync, and deterministic analytics.

## Run with Docker Compose

```bash
docker compose up --build
```

Apply migrations:

```bash
docker compose run --rm migrate
```

## Local env

- `DATABASE_URL` (default `postgres://baseline:baseline@localhost:5432/baseline?sslmode=disable`)
- `API_TOKEN` (default `baseline-dev-token`)
- `DEFAULT_USER_ID` (default `00000000-0000-0000-0000-000000000001`)
- `PORT` (default `8080`)

## Migrations

SQL files live under `migrations/`:
- `001_raw_tables.*.sql`
- `002_projection_tables.*.sql`

Runner:

```bash
go run ./cmd/migrate
```
