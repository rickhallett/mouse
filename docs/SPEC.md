# Mouse v1 Spec

## Goal
Minimal, terminal-first agent runtime with strong defaults: Telegram only, Docker-only sandbox, Markdown persistence, SQLite + vector indexing, and fast boot.

## Components
- Gateway (Go): HTTP + WebSocket API, session routing, approvals, cron.
- Telegram Adapter: Webhook receiver, outbound sender (v1 only inbound + log).
- Storage: Markdown source of truth; SQLite mirror + index.
- Indexer: Background worker to sync Markdown -> SQLite + vector.
- Sandbox Runner: Docker exec with workspace-only RW.
- CLI: Start/stop/status/logs/run/check.

## Data Model
- runtime/memory/: Markdown knowledge base
- runtime/sessions/: Markdown sessions
- runtime/notes/: Scratch + summaries
- runtime/sqlite/: State DB + vector index
- runtime/logs/: JSONL logs

## Security Defaults
- Telegram allowlist required
- Webhook secret supported
- No channel-driven config writes
- Docker mandatory, host exec disabled
- Workspace is the only RW mount

## Gate
- `mouse -check -config ./config/mouse.yaml` validates config and exits 0 on success
- `scripts/gate.sh` runs build + check
