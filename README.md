# mouse

Mouse is a terminal-first agent runtime for teams that want Telegram as the only chat surface and Docker as the only execution surface. It is opinionated about auditability (JSONL logs), persistence (Markdown as source of truth), and safety (workspace-only RW mounts, network-none containers).

**Who This Is For**
- Ops/infra teams running a self-hosted agent runtime for a small number of trusted Telegram users.
- Developers who need an auditable, reproducible execution surface (Docker only) with a minimal API.
- Teams that want Markdown and SQLite as canonical data stores instead of opaque databases.

**What It Does**
- Ingests Telegram updates (webhook) and appends them to Markdown sessions.
- Calls an LLM for responses and persists results to Markdown + SQLite.
- Runs tools inside Docker with allow/deny policy enforcement.
- Indexes Markdown files and exposes basic search over them.
- Schedules cron jobs that post to sessions.

**What It Does Not Do**
- Multi-channel chat or multi-tenant isolation.
- Direct host command execution (intentionally blocked).
- Rich LLM tool planning or agent orchestration beyond session append + reply.

**Quick Start (Local)**
1. Build and validate config:
```bash
go build -o bin/mouse ./cmd/mouse
TELEGRAM_BOT_TOKEN=dummy TELEGRAM_WEBHOOK_SECRET=dummy ANTHROPIC_API_KEY=dummy ./bin/mouse -check -config ./config/mouse.yaml
```
1. Run the gateway:
```bash
./bin/mouse -config ./config/mouse.yaml -addr :8080
```

**Configuration Notes**
- `config/mouse.yaml` uses `env:VAR_NAME` for secrets.
- `telegram.allow_from` is the primary allowlist for inbound and outbound.
- `sandbox.docker.binds` should include exactly one RW workspace mount.
- `index.watch.paths` is what the indexer scans.
- Cron schedules currently accept `minute hour * * *` (minute/hour only).

**HTTP Endpoints**
- `GET /health`
- `POST /telegram-webhook` (configurable path)
- `POST /tools/run`
- `GET /index/search?q=...&limit=...`
- `POST /index/reindex`
- `POST /approvals/submit`

**Data Layout**
- `runtime/sessions/` Markdown sessions
- `runtime/memory/` Markdown memory store
- `runtime/sqlite/` SQLite DB + index tables
- `runtime/logs/` JSONL logs (`mouse.log`)

**CLI (mousectl)**
- `mousectl status -addr http://localhost:8080`
- `mousectl run -tool read -- ls -la`
- `mousectl reindex -addr http://localhost:8080`
- `mousectl search -q "project status" -limit 5`
- `mousectl approve <id>`
- `mousectl logs -file ./runtime/logs/mouse.log -n 100`

**Fly.io Deploy**
- Create volume: `./scripts/fly/volume-setup.sh`
- Deploy + set webhook: `./scripts/fly/deploy.sh`
- Webhook script expects `TELEGRAM_BOT_TOKEN`, `TELEGRAM_WEBHOOK_PUBLIC_URL`, `TELEGRAM_WEBHOOK_PATH`, and optional `TELEGRAM_WEBHOOK_SECRET`.

**Security Model (Summary)**
- Telegram allowlist is enforced for inbound and outbound.
- Webhook secret token is supported.
- All tools execute inside Docker with `--read-only` root and `--network none`.
- Workspace is the only RW mount.

**Development**
- `./scripts/gate.sh` runs build + config check + tests.
- `go test ./...`
