# Settle backend

Go API (`github.com/Firakef1/settle/backend`).

## Setup

```bash
# from repo root
docker compose up -d
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/settle
```

API base (planned): `http://localhost:8080/v1`

## Structure

See `docs/PRD.md` (Backend Architecture).
