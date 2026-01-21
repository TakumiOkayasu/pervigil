# Pervigil

VyOSルーター向け監視スクリプト集。Discord Webhookで通知。

> *Pervigil* (ラテン語): 常に見守る者

## 構成

```
scripts/
├── discord-notify.sh   # Discord通知基盤
├── nic-monitor.sh      # NIC温度監視
├── log-monitor.sh      # ログ監視 (エラー/警告抽出)
└── .env                # シークレット (要作成)
```

## 機能

### NIC温度監視 (nic-monitor.sh)

| 温度 | 状態 | アクション |
|------|------|-----------|
| <70℃ | 正常 | - |
| 70-85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | Discord通知 + 速度1Gbps制限 |
| <65℃ (復旧) | 正常 | 速度制限解除 |

### ログ監視 (log-monitor.sh)

- syslogからエラー/警告を抽出
- `/config/logs/errors.log` に永続保存 (最大10MB)
- エラー検出時にDiscord通知
- 既知のノイズは除外

## VyOSへのデプロイ

### 1. ファイル転送

```bash
[Mac] scp -r scripts/ vyos@192.168.1.1:/config/scripts/
```

### 2. 権限設定

```bash
[VyOS] chmod +x /config/scripts/*.sh
```

### 3. 環境変数設定

```bash
[VyOS] cat > /config/scripts/.env << 'EOF'
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
NIC_INTERFACE="eth1"
EOF
chmod 600 /config/scripts/.env
```

### 4. 動作テスト

```bash
[VyOS] /config/scripts/nic-monitor.sh --test
[VyOS] /config/scripts/log-monitor.sh --status
```

### 5. cron設定

```bash
[VyOS] configure
set system task-scheduler task nic-monitor interval 5m
set system task-scheduler task nic-monitor executable path /config/scripts/nic-monitor.sh

set system task-scheduler task log-monitor interval 1m
set system task-scheduler task log-monitor executable path /config/scripts/log-monitor.sh

commit
save
```

## コマンドオプション

### nic-monitor.sh

| オプション | 説明 |
|-----------|------|
| (なし) | 温度監視実行 |
| --test | 温度読み取りテスト |
| --status | 現在の温度と状態表示 |

### log-monitor.sh

| オプション | 説明 |
|-----------|------|
| (なし) | ログ監視実行 |
| --status | 永続ログの状態表示 |
| --tail | 永続ログをtail -f |

## Discord Webhook設定

1. Discordサーバー設定 → 連携サービス → Webhook
2. 「新しいウェブフック」を作成
3. URLをコピーして `.env` に設定

## 注意事項

- `/config/` 以下はVyOS再起動後も永続化される
- 温度取得はIntel X540-T2 (ixgbe) を想定
- sensorsコマンドがない場合は `/sys/class/hwmon` を使用
