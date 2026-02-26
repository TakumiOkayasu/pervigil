# Pervigil

VyOSルーター向け監視ツール。Discord Webhookで通知。

## 構成

```text
bot/
├── cmd/
│   ├── pervigil-bot/       # Discord Bot
│   └── pervigil-monitor/   # 監視デーモン
└── internal/
    ├── anthropic/          # Anthropic Admin APIクライアント
    ├── config/             # 設定読み込み
    ├── handler/            # Botコマンドハンドラ
    ├── sysinfo/            # システム情報取得
    ├── temperature/        # 温度センサー
    ├── notifier/           # Discord Webhook通知
    └── monitor/            # NIC/ログ/コスト監視ロジック
```

## ビルド

Dockerによるクロスコンパイル。成果物は `bin/` に出力。

```bash
cd bot
docker build -f Dockerfile.build -t pervigil-builder .
docker create --name tmp-pervigil pervigil-builder
docker cp tmp-pervigil:/pervigil-monitor ../bin/
docker cp tmp-pervigil:/pervigil-bot ../bin/
docker rm tmp-pervigil
```

### ビルドオプション

| 変数 | デフォルト | 説明 |
| ------ | ----------- | ------ |
| TARGETOS | linux | ターゲットOS |
| TARGETARCH | amd64 | ターゲットアーキテクチャ |

```bash
# ARM64向けビルド
docker build --build-arg TARGETARCH=arm64 -f Dockerfile.build -t pervigil-builder .
```

## デプロイ

```bash
# VyOSへ転送
scp bin/pervigil-monitor bin/pervigil-bot vyos@<IP>:/config/pervigil/

# .env作成 (VyOS上)
cat > /config/pervigil/.env << 'EOF'
DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/..."
NIC_INTERFACE="eth1"
BOT_TOKEN="your-discord-bot-token"
ANTHROPIC_ADMIN_KEY="your-anthropic-admin-key"
EOF

# 起動
cd /config/pervigil && ./pervigil-monitor
```

### systemd化

```bash
# serviceファイル転送
scp deploy/*.service vyos@<IP>:/tmp/

# VyOS上で実行
sudo mv /tmp/pervigil-*.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now pervigil-monitor
sudo systemctl enable --now pervigil-bot
```

確認:

```bash
systemctl status pervigil-monitor pervigil-bot
```

## 監視デーモン (pervigil-monitor)

### 機能

| 機能 | 説明 |
| ------ | ------ |
| NIC温度監視 | FSMベースの状態管理、速度制限制御 |
| ログ監視 | パターンマッチ、除外ルール対応 |
| コスト監視 | Anthropic API利用コスト監視、予算閾値アラート |
| Discord通知 | Webhook経由でリアルタイム通知 |

### NIC温度閾値

| 温度 | 状態 | アクション |
| ------ | ------ | ------------ |
| <70℃ | 正常 | - |
| 70-85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | Discord通知 + 速度1Gbps制限 |
| <65℃ (復旧) | 正常 | 速度制限解除 |

### 環境変数（monitor）

| 変数 | 必須 | デフォルト | 説明 |
| ------ | ------ | ----------- | ------ |
| DISCORD_WEBHOOK_URL | Yes | - | Webhook URL |
| NIC_INTERFACE | No | eth1 | 監視NIC |
| CHECK_INTERVAL | No | 60 | チェック間隔(秒) |
| STATE_FILE | No | /tmp/pervigil-state | 状態ファイル |
| LOG_FILE | No | /var/log/syslog | 監視ログ |
| ANTHROPIC_ADMIN_KEY | No | - | Anthropic Admin APIキー |
| COST_CHECK_INTERVAL | No | 3600 | コストチェック間隔(秒) |
| DAILY_BUDGET_WARN | No | 5.0 | 日次警告閾値($) |
| DAILY_BUDGET_CRIT | No | 10.0 | 日次危険閾値($) |
| COST_STATE_FILE | No | /tmp/pervigil-cost-state | コスト状態ファイル |
| ERROR_SUPPRESS_INTERVAL | No | 3600 | エラー抑制間隔(秒) |

## Discord Bot (pervigil-bot)

### コマンド一覧

| コマンド | 説明 |
| ---------- | ------ |
| /nic | NIC温度を表示 |
| /temp | 全温度情報を表示 (CPU + NIC) |
| /status | システム状態サマリー |
| /cpu | CPU使用率とロードアベレージを表示 |
| /memory | メモリ使用状況を表示 |
| /disk | ディスク使用状況を表示 |
| /info | ルーター全情報を表示 |
| /network | 全NIC情報を表示 |
| /claude | Claude API利用状況を表示 |

### 環境変数 (Bot)

| 変数 | 必須 | 説明 |
| ------ | ------ | ------ |
| BOT_TOKEN | Yes | Discord Bot Token |
| GUILD_ID | No | サーバーID (コマンド即時反映用) |
| ANTHROPIC_ADMIN_KEY | No | Anthropic Admin APIキー |
| DAILY_BUDGET_WARN | No | 日次警告閾値($) |
| DAILY_BUDGET_CRIT | No | 日次危険閾値($) |

## 注意事項

- `/config/` 以下はVyOS再起動後も永続化
- 温度取得はIntel X540-T2 (ixgbe) を想定
- `.env` は実行ディレクトリに配置
