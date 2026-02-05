# Pervigil

VyOSルーター向け監視ツール。Discord Webhookで通知。

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
|------|-----------|------|
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

### 環境変数

| 変数 | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| DISCORD_WEBHOOK_URL | Yes | - | Webhook URL |
| NIC_INTERFACE | No | eth1 | 監視NIC |
| CHECK_INTERVAL | No | 60 | チェック間隔(秒) |
| STATE_FILE | No | /tmp/pervigil-state | 状態ファイル |
| LOG_FILE | No | /var/log/syslog | 監視ログ |

## Discord Bot (pervigil-bot)

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

### 環境変数

| 変数 | 必須 | 説明 |
|------|------|------|
| BOT_TOKEN | Yes | Discord Bot Token |
| GUILD_ID | No | サーバーID (コマンド即時反映用) |

## 注意事項

- `/config/` 以下はVyOS再起動後も永続化
- 温度取得はIntel X540-T2 (ixgbe) を想定
- `.env` は実行ディレクトリに配置
