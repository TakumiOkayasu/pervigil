#!/bin/bash
# NIC温度監視スクリプト
# 段階的対応: 警告(70-85℃)→Discord通知、危険(>85℃)→速度制限+通知

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/discord-notify.sh"

# 設定
INTERFACE="${NIC_INTERFACE:-eth1}"  # 監視対象NIC
TEMP_WARNING=70                      # 警告閾値 (℃)
TEMP_CRITICAL=85                     # 危険閾値 (℃)
TEMP_RECOVERY=65                     # 復旧閾値 (℃)
STATE_FILE="/tmp/nic-monitor-state"  # 状態ファイル

# 状態定数
STATE_NORMAL="normal"
STATE_WARNING="warning"
STATE_CRITICAL="critical"

# NIC温度取得 (Intel X540-T2)
get_nic_temp() {
    local temp

    # 方法1: sensorsコマンド (lm-sensorsインストール済みの場合)
    if command -v sensors &>/dev/null; then
        temp=$(sensors 2>/dev/null | grep -i "ixgbe" -A5 | grep -i "temp" | head -1 | awk '{print $2}' | tr -d '+°C')
        if [[ -n "$temp" ]]; then
            echo "$temp"
            return 0
        fi
    fi

    # 方法2: /sys/class/hwmon から取得
    for hwmon in /sys/class/hwmon/hwmon*/; do
        local name
        name=$(cat "${hwmon}name" 2>/dev/null || echo "")
        if [[ "$name" == "ixgbe" ]] || [[ "$name" == "coretemp" ]]; then
            local temp_file="${hwmon}temp1_input"
            if [[ -f "$temp_file" ]]; then
                temp=$(cat "$temp_file")
                echo $((temp / 1000))  # ミリ度→度
                return 0
            fi
        fi
    done

    # 方法3: ethtoolのモジュール温度 (SFP+の場合)
    if command -v ethtool &>/dev/null; then
        temp=$(ethtool -m "$INTERFACE" 2>/dev/null | grep -i "module temperature" | awk '{print $NF}' | tr -d 'C')
        if [[ -n "$temp" ]]; then
            echo "$temp"
            return 0
        fi
    fi

    echo "N/A"
    return 1
}

# 現在の状態取得
get_state() {
    if [[ -f "$STATE_FILE" ]]; then
        cat "$STATE_FILE"
    else
        echo "$STATE_NORMAL"
    fi
}

# 状態保存
set_state() {
    echo "$1" > "$STATE_FILE"
}

# NIC速度制限 (1Gbpsに下げる)
limit_nic_speed() {
    if command -v ethtool &>/dev/null; then
        ethtool -s "$INTERFACE" speed 1000 duplex full autoneg off 2>/dev/null || true
        echo "[INFO] NIC speed limited to 1Gbps"
    fi
}

# NIC速度復旧 (自動ネゴシエーション)
restore_nic_speed() {
    if command -v ethtool &>/dev/null; then
        ethtool -s "$INTERFACE" autoneg on 2>/dev/null || true
        echo "[INFO] NIC speed restored to auto-negotiation"
    fi
}

# メイン処理
main() {
    local temp
    temp=$(get_nic_temp)

    if [[ "$temp" == "N/A" ]]; then
        echo "[WARN] Could not read NIC temperature"
        return 1
    fi

    local current_state
    current_state=$(get_state)
    local new_state="$STATE_NORMAL"
    local hostname
    hostname=$(hostname)

    echo "[INFO] NIC temperature: ${temp}°C (state: $current_state)"

    # 温度に応じた処理
    if [[ "$temp" -ge "$TEMP_CRITICAL" ]]; then
        new_state="$STATE_CRITICAL"
        if [[ "$current_state" != "$STATE_CRITICAL" ]]; then
            # 危険状態に遷移
            local fields='[{"name":"Temperature","value":"'"${temp}°C"'","inline":true},{"name":"Threshold","value":"'"${TEMP_CRITICAL}°C"'","inline":true},{"name":"Action","value":"Speed limited to 1Gbps","inline":true}]'
            send_discord "🔥 NIC過熱警報 - $hostname" "NIC温度が危険域に達しました。速度を1Gbpsに制限します。" "red" "$fields"
            limit_nic_speed
        fi

    elif [[ "$temp" -ge "$TEMP_WARNING" ]]; then
        new_state="$STATE_WARNING"
        if [[ "$current_state" == "$STATE_NORMAL" ]]; then
            # 警告状態に遷移
            local fields='[{"name":"Temperature","value":"'"${temp}°C"'","inline":true},{"name":"Warning Threshold","value":"'"${TEMP_WARNING}°C"'","inline":true},{"name":"Critical Threshold","value":"'"${TEMP_CRITICAL}°C"'","inline":true}]'
            send_discord "⚠️ NIC温度警告 - $hostname" "NIC温度が警告域に達しました。監視を継続します。" "yellow" "$fields"
        fi

    else
        new_state="$STATE_NORMAL"
        if [[ "$current_state" == "$STATE_CRITICAL" && "$temp" -le "$TEMP_RECOVERY" ]]; then
            # 危険状態から復旧
            local fields='[{"name":"Temperature","value":"'"${temp}°C"'","inline":true},{"name":"Action","value":"Speed restored to auto","inline":true}]'
            send_discord "✅ NIC温度正常化 - $hostname" "NIC温度が正常範囲に戻りました。速度制限を解除します。" "green" "$fields"
            restore_nic_speed
        elif [[ "$current_state" == "$STATE_WARNING" ]]; then
            # 警告状態から復旧
            local fields='[{"name":"Temperature","value":"'"${temp}°C"'","inline":true}]'
            send_discord "✅ NIC温度正常化 - $hostname" "NIC温度が正常範囲に戻りました。" "green" "$fields"
        fi
    fi

    set_state "$new_state"
}

# 引数処理
case "${1:-}" in
    --test)
        echo "Testing NIC temperature reading..."
        temp=$(get_nic_temp)
        echo "Temperature: ${temp}°C"
        ;;
    --status)
        temp=$(get_nic_temp)
        state=$(get_state)
        echo "Temperature: ${temp}°C"
        echo "State: $state"
        ;;
    *)
        main
        ;;
esac
