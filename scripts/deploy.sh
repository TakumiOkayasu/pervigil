#!/bin/bash
set -e

# Usage: ./scripts/deploy.sh <VYOS_HOST>
# Example: ./scripts/deploy.sh 192.168.1.1

VYOS_HOST="${1:-}"
if [ -z "$VYOS_HOST" ]; then
    echo "Usage: $0 <VYOS_HOST>"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "==> Building..."
cd "$ROOT_DIR/bot"
docker build --no-cache -f Dockerfile.build -t pervigil-builder .
docker create --name tmp-pervigil pervigil-builder
mkdir -p "$ROOT_DIR/bin"
docker cp tmp-pervigil:/pervigil-monitor "$ROOT_DIR/bin/"
docker cp tmp-pervigil:/pervigil-bot "$ROOT_DIR/bin/"
docker rm tmp-pervigil

echo "==> Transferring binaries..."
scp "$ROOT_DIR/bin/pervigil-monitor" "$ROOT_DIR/bin/pervigil-bot" "vyos@${VYOS_HOST}:/config/pervigil/"

echo "==> Transferring service files..."
scp "$ROOT_DIR/deploy/pervigil-monitor.service" "$ROOT_DIR/deploy/pervigil-bot.service" "vyos@${VYOS_HOST}:/tmp/"

echo "==> Installing services on VyOS..."
ssh "vyos@${VYOS_HOST}" 'sudo mv /tmp/pervigil-*.service /etc/systemd/system/ && sudo systemctl daemon-reload'

echo "==> Restarting services..."
ssh "vyos@${VYOS_HOST}" 'sudo systemctl restart pervigil-monitor pervigil-bot'

echo "==> Done! Checking status..."
ssh "vyos@${VYOS_HOST}" 'systemctl status pervigil-monitor pervigil-bot --no-pager'
