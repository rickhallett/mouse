#!/bin/sh
set -e

echo "[gate] build"
go build -o bin/mouse ./cmd/mouse

echo "[gate] config check"
TELEGRAM_BOT_TOKEN=dummy \
TELEGRAM_WEBHOOK_SECRET=dummy \
ANTHROPIC_API_KEY=dummy \
./bin/mouse -check -config ./config/mouse.yaml

echo "[gate] ok"
