#!/bin/bash
# Discord Webhook通知スクリプト
# 使用方法: source discord-notify.sh && send_discord "タイトル" "メッセージ" "color"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

# 環境変数読み込み
if [[ -f "$ENV_FILE" ]]; then
    source "$ENV_FILE"
fi

# 色定義 (10進数)
COLOR_GREEN=5763719    # 0x57f287
COLOR_YELLOW=16776960  # 0xffff00
COLOR_RED=15548997     # 0xed4245
COLOR_BLUE=5793266     # 0x5865f2

# Discord Webhook送信
# 引数: $1=タイトル, $2=メッセージ, $3=色(green/yellow/red/blue), $4=フィールド(JSON配列,省略可)
send_discord() {
    local title="$1"
    local message="$2"
    local color_name="${3:-blue}"
    local fields="${4:-[]}"

    if [[ -z "$DISCORD_WEBHOOK_URL" ]]; then
        echo "[ERROR] DISCORD_WEBHOOK_URL is not set" >&2
        return 1
    fi

    # 色名から数値に変換
    local color
    case "$color_name" in
        green)  color=$COLOR_GREEN ;;
        yellow) color=$COLOR_YELLOW ;;
        red)    color=$COLOR_RED ;;
        *)      color=$COLOR_BLUE ;;
    esac

    # タイムスタンプ
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # ペイロード作成
    local payload
    payload=$(cat <<EOF
{
  "username": "Pervigil",
  "embeds": [{
    "title": "$title",
    "description": "$message",
    "color": $color,
    "fields": $fields,
    "timestamp": "$timestamp"
  }]
}
EOF
)

    # 送信
    local response
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json; charset=utf-8" \
        -d "$payload" \
        "$DISCORD_WEBHOOK_URL")

    local http_code
    http_code=$(echo "$response" | tail -n1)

    if [[ "$http_code" -ge 200 && "$http_code" -lt 300 ]]; then
        return 0
    else
        echo "[ERROR] Discord API error: $http_code" >&2
        return 1
    fi
}

# テスト用関数
test_discord() {
    send_discord "テスト通知" "Pervigilからのテストメッセージです。" "blue"
}
