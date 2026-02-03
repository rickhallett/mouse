#!/bin/sh
set -e

if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
  echo "TELEGRAM_BOT_TOKEN is required" >&2
  exit 1
fi
if [ -z "$TELEGRAM_WEBHOOK_PUBLIC_URL" ]; then
  echo "TELEGRAM_WEBHOOK_PUBLIC_URL is required" >&2
  exit 1
fi
if [ -z "$TELEGRAM_WEBHOOK_PATH" ]; then
  echo "TELEGRAM_WEBHOOK_PATH is required" >&2
  exit 1
fi

SECRET_PAYLOAD=""
if [ -n "$TELEGRAM_WEBHOOK_SECRET" ]; then
  SECRET_PAYLOAD=", \"secret_token\": \"$TELEGRAM_WEBHOOK_SECRET\""
fi

curl -sS -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
  -H "content-type: application/json" \
  -d "{\"url\": \"${TELEGRAM_WEBHOOK_PUBLIC_URL}${TELEGRAM_WEBHOOK_PATH}\"${SECRET_PAYLOAD}}"
