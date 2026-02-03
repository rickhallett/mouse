# Security Model

## Assumptions
- Single channel: Telegram
- Host runs Docker
- Workspace volume is the only writable mount

## Inbound Controls
- Allowlist enforced for Telegram sender IDs/usernames
- Optional webhook secret token

## Execution Controls
- All tool execution runs in Docker
- Host exec disabled (no direct system commands)
- Workspace bind is read-write; container root is read-only
- Network defaults to none (tightest possible)

## Persistence
- Markdown files are source of truth
- SQLite mirrors data for indexing and retrieval
- Secrets are never written to Markdown

## Operational Hardening
- Use environment variables or secret files
- Keep runtime on its own volume
- Rotate logs regularly
