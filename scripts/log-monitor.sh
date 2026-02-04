#!/bin/bash
# ログ監視スクリプト
# エラー/警告をDiscordに通知し、永続ログに保存

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/discord-notify.sh"

# 設定
LOG_DIR="/var/log"
PERSISTENT_LOG="/config/logs/errors.log"     # 永続保存するエラーログ
LAST_POS_FILE="/tmp/log-monitor-pos"         # 最終読み取り位置
MAX_PERSISTENT_SIZE=10485760                  # 永続ログ最大サイズ (10MB)
MAX_LINES_PER_RUN=100                         # 1回の実行で処理する最大行数

# 監視対象パターン
ERROR_PATTERNS=(
    "error"
    "ERROR"
    "failed"
    "FAILED"
    "critical"
    "CRITICAL"
    "panic"
    "PANIC"
)

WARNING_PATTERNS=(
    "warning"
    "WARNING"
    "warn"
    "WARN"
)

# 除外パターン (ノイズ除去)
EXCLUDE_PATTERNS=(
    "DHCP4_BUFFER_RECEIVE_FAIL.*Truncated"  # 既知の問題
    "netlink-dp.*Network is down"            # 一時的なネットワーク状態
    "pam_unix.*authentication failure"       # SSH試行
)

# 永続ログディレクトリ作成
ensure_log_dir() {
    local dir
    dir=$(dirname "$PERSISTENT_LOG")
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
    fi
}

# ログローテーション (永続ログ)
rotate_persistent_log() {
    if [[ -f "$PERSISTENT_LOG" ]]; then
        local size
        size=$(stat -f%z "$PERSISTENT_LOG" 2>/dev/null || stat -c%s "$PERSISTENT_LOG" 2>/dev/null || echo 0)
        if [[ "$size" -gt "$MAX_PERSISTENT_SIZE" ]]; then
            mv "$PERSISTENT_LOG" "${PERSISTENT_LOG}.1"
            echo "[INFO] Rotated persistent log"
        fi
    fi
}

# パターンマッチ判定
matches_pattern() {
    local line="$1"
    shift
    local patterns=("$@")

    for pattern in "${patterns[@]}"; do
        if echo "$line" | grep -qiE "$pattern"; then
            return 0
        fi
    done
    return 1
}

# 除外判定
should_exclude() {
    local line="$1"
    for pattern in "${EXCLUDE_PATTERNS[@]}"; do
        if echo "$line" | grep -qE "$pattern"; then
            return 0
        fi
    done
    return 1
}

# syslogの新しいエントリを取得
get_new_log_entries() {
    local log_file="${LOG_DIR}/syslog"
    local last_pos=0

    if [[ -f "$LAST_POS_FILE" ]]; then
        last_pos=$(cat "$LAST_POS_FILE")
    fi

    if [[ ! -f "$log_file" ]]; then
        echo ""
        return
    fi

    local current_size
    current_size=$(stat -f%z "$log_file" 2>/dev/null || stat -c%s "$log_file" 2>/dev/null || echo 0)

    # ログローテーションが発生した場合
    if [[ "$current_size" -lt "$last_pos" ]]; then
        last_pos=0
    fi

    # 新しいエントリを取得
    tail -c +$((last_pos + 1)) "$log_file" 2>/dev/null | head -n "$MAX_LINES_PER_RUN"

    # 位置を更新
    echo "$current_size" > "$LAST_POS_FILE"
}

# メイン処理
main() {
    ensure_log_dir
    rotate_persistent_log

    local hostname
    hostname=$(hostname)
    local new_entries
    new_entries=$(get_new_log_entries)

    if [[ -z "$new_entries" ]]; then
        return 0
    fi

    local error_count=0
    local warning_count=0
    local error_lines=""
    local warning_lines=""

    while IFS= read -r line; do
        [[ -z "$line" ]] && continue

        # 除外パターンチェック
        if should_exclude "$line"; then
            continue
        fi

        # エラー判定
        if matches_pattern "$line" "${ERROR_PATTERNS[@]}"; then
            ((error_count++)) || true
            error_lines+="$line"$'\n'
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] [ERROR] $line" >> "$PERSISTENT_LOG"
            continue
        fi

        # 警告判定
        if matches_pattern "$line" "${WARNING_PATTERNS[@]}"; then
            ((warning_count++)) || true
            warning_lines+="$line"$'\n'
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] [WARN] $line" >> "$PERSISTENT_LOG"
        fi
    done <<< "$new_entries"

    # Discord通知 (エラーがある場合)
    if [[ "$error_count" -gt 0 ]]; then
        local truncated_errors
        truncated_errors=$(echo "$error_lines" | head -c 1000)
        local fields='[{"name":"Error Count","value":"'"$error_count"'","inline":true},{"name":"Log File","value":"'"$PERSISTENT_LOG"'","inline":true}]'
        send_discord "🚨 ログエラー検出 - $hostname" "\`\`\`\n${truncated_errors}\n\`\`\`" "red" "$fields"
    fi

    # Discord通知 (警告のみの場合、エラーがなければ)
    if [[ "$warning_count" -gt 0 && "$error_count" -eq 0 ]]; then
        # 警告は5件以上まとめて通知 (ノイズ軽減)
        if [[ "$warning_count" -ge 5 ]]; then
            local fields='[{"name":"Warning Count","value":"'"$warning_count"'","inline":true}]'
            send_discord "⚠️ ログ警告 - $hostname" "過去の監視期間に${warning_count}件の警告が検出されました。" "yellow" "$fields"
        fi
    fi

    echo "[INFO] Processed: $error_count errors, $warning_count warnings"
}

# テスト用エントリポイント（DI対応）
run_test() {
    echo "Testing log patterns..."
    if matches_pattern "ERROR test" "${ERROR_PATTERNS[@]}"; then
        echo "ERROR pattern works"
    fi
}

run_status() {
    echo "Persistent log: $PERSISTENT_LOG"
    if [[ -f "$PERSISTENT_LOG" ]]; then
        local size
        size=$(stat -f%z "$PERSISTENT_LOG" 2>/dev/null || stat -c%s "$PERSISTENT_LOG" 2>/dev/null || echo 0)
        echo "Size: $size bytes"
        echo "Last 5 entries:"
        tail -5 "$PERSISTENT_LOG"
    else
        echo "No persistent log yet"
    fi
}

run_tail() {
    if [[ -f "$PERSISTENT_LOG" ]]; then
        tail -f "$PERSISTENT_LOG"
    else
        echo "No persistent log yet"
    fi
}

# 直接実行時のみ引数処理
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        --test)
            run_test
            ;;
        --status)
            run_status
            ;;
        --tail)
            run_tail
            ;;
        *)
            main
            ;;
    esac
fi
