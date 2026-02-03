# mouse

Minimal, terminal-first agent runtime. Telegram-only, Docker-mandatory, Markdown persistence with SQLite + vector indexing.

## Quick start

```bash
go build -o bin/mouse ./cmd/mouse
TELEGRAM_BOT_TOKEN=dummy ANTHROPIC_API_KEY=dummy ./bin/mouse -check -config ./config/mouse.yaml
./bin/mouse -config ./config/mouse.yaml -addr :8080
```

## Gate

```bash
./scripts/gate.sh
```

## Structure
- `cmd/mouse`: gateway binary
- `internal/`: core packages
- `runtime/`: data directories (Markdown + SQLite + logs)
- `config/`: config templates
- `scripts/`: Docker + Fly.io helpers
