# Pervigil

VyOSルーター向け監視スクリプト集。Discord Webhookで通知。

> *Pervigil* (ラテン語): 常に見守る者

## 構成

```text
bot/
├── cmd/
│   ├── pervigil-bot/       # Discord Bot
│   └── pervigil-monitor/   # 監視デーモン
└── internal/
    ├── config/             # 設定読み込み
    ├── handler/            # Botコマンドハンドラ
    ├── sysinfo/            # システム情報取得
    ├── temperature/        # 温度センサー
    ├── notifier/           # Discord Webhook通知
    └── monitor/            # NIC/ログ監視ロジック
```

## 監視デーモン (pervigil-monitor)

NIC温度監視・ログ監視を行う単一バイナリ。

### 機能

| 機能 | 説明 |
|------|------|
| NIC温度監視 | FSMベースの状態管理、速度制限制御 |
| ログ監視 | パターンマッチ、除外ルール対応 |
| Discord通知 | Webhook経由でリアルタイム通知 |

### NIC温度閾値

| 温度 | 状態 | アクション |
|------|------|------------|
| <70℃ | 正常 | - |
| 70-85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | Discord通知 + 速度1Gbps制限 |
| <65℃ (復旧) | 正常 | 速度制限解除 |

### ビルド

```bash
cd bot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-monitor ./cmd/pervigil-monitor
```

### 環境変数

| 変数 | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| DISCORD_WEBHOOK_URL | Yes | - | Webhook URL |
| NIC_INTERFACE | No | eth1 | 監視NIC |
| CHECK_INTERVAL | No | 60 | チェック間隔(秒) |
| STATE_FILE | No | /tmp/pervigil-state | 状態ファイル |
| LOG_FILE | No | /var/log/syslog | 監視ログ |

### デプロイ

```bash
# ビルド
cd bot && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-monitor ./cmd/pervigil-monitor

# 転送
scp bot/pervigil-monitor vyos@192.168.x.x:/config/scripts/pervigil/

# .env作成 (VyOS上)
cat > /config/scripts/pervigil/.env << 'EOF'
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
NIC_INTERFACE="eth1"
EOF

# 起動
cd /config/scripts/pervigil && ./pervigil-monitor
```

### systemdサービス化 (オプション)

```bash
# /etc/systemd/system/pervigil-monitor.service
[Unit]
Description=Pervigil Monitor
After=network.target

[Service]
Type=simple
WorkingDirectory=/config/scripts/pervigil
ExecStart=/config/scripts/pervigil/pervigil-monitor
Restart=always

[Install]
WantedBy=multi-user.target
```

## Discord Bot (pervigil-bot)

Discordからオンデマンドでシステム情報を取得するBot。

### コマンド一覧

| コマンド | 説明 |
|----------|------|
| /nic | NIC温度を表示 |
| /temp | 全温度情報を表示 (CPU + NIC) |
| /status | システム状態サマリー |
| /cpu | CPU使用率とロードアベレージ |
| /memory | メモリ使用状況 |
| /network | 全NIC情報 (状態/速度/統計) |
| /disk | ディスク使用状況 |
| /info | ルーター全情報 |

### ビルド・デプロイ

```bash
cd bot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-bot ./cmd/pervigil-bot
scp bot/pervigil-bot vyos@192.168.x.x:/config/scripts/pervigil/
```

### 環境変数 (bot/.env)

```bash
BOT_TOKEN="your-discord-bot-token"
GUILD_ID="your-guild-id"           # 省略可 (コマンド即時反映用)
MONITOR_NICS="eth0,eth1,eth2"      # 省略可 (監視NIC一覧)
```

## Discord Webhook設定

1. Discordサーバー設定 → 連携サービス → Webhook
2. 「新しいウェブフック」を作成
3. URLをコピーして `.env` に設定

## 注意事項

- `/config/` 以下はVyOS再起動後も永続化される
- 温度取得はIntel X540-T2 (ixgbe) を想定
- sensorsコマンドがない場合は `/sys/class/hwmon` を使用
