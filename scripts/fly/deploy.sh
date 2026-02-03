#!/bin/sh
set -e
fly deploy -c scripts/fly/fly.toml
./scripts/fly/webhook.sh
