# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

Pervigil: VyOSルーター向け監視スクリプト集。Discord Webhookで通知。

## アーキテクチャ

```text
scripts/
├── discord-notify.sh   # 通知基盤 (他スクリプトからsource)
├── nic-monitor.sh      # NIC温度監視 → discord-notify.sh依存
├── log-monitor.sh      # syslogエラー監視 → discord-notify.sh依存
└── .env                # シークレット (DISCORD_WEBHOOK_URL, NIC_INTERFACE)

bot/                    # Discord Bot (Go)
├── cmd/pervigil-bot/   # エントリーポイント
├── internal/
│   ├── config/         # 設定読み込み
│   ├── handler/        # コマンドハンドラ
│   └── temp/           # 温度センサー
├── go.mod
└── go.sum
```

**依存関係**: `nic-monitor.sh`, `log-monitor.sh` → `source discord-notify.sh`

## 開発コマンド

### ローカルテスト

```bash
# Discord通知テスト (.env設定後)
cd scripts && source discord-notify.sh && test_discord

# NIC温度読み取りテスト
./scripts/nic-monitor.sh --test

# 現在状態確認
./scripts/nic-monitor.sh --status
./scripts/log-monitor.sh --status
```

### VyOSデプロイ

```bash
scp -r scripts/ vyos@<IP>:/config/scripts/
```

## 環境変数

| 変数 | 必須 | 用途 |
|------|------|------|
| DISCORD_WEBHOOK_URL | ✅ | Discord通知先 |
| NIC_INTERFACE | - | 監視NIC (default: eth1) |

## 閾値設定 (nic-monitor.sh)

| 温度 | 状態 | アクション |
|------|------|------------|
| <70℃ | 正常 | - |
| 70-85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | 速度1Gbps制限 |
| <65℃ | 復旧 | 速度制限解除 |

## Discord Bot (pervigil-bot)

### ビルド

```bash
cd bot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-bot ./cmd/pervigil-bot
```

### デプロイ

```bash
scp bot/pervigil-bot vyos@<IP>:/config/scripts/pervigil/
```

### 起動

```bash
# VyOS上
cd /config/scripts/pervigil
./pervigil-bot
```

### 環境変数 (bot用)

| 変数 | 必須 | 用途 |
|------|------|------|
| BOT_TOKEN | ✅ | Discord Bot Token |
| GUILD_ID | - | サーバーID (コマンド即時反映用) |
| CLEANUP_COMMANDS | - | 終了時コマンド削除 (1で有効) |

`.env` は実行ファイルと同じディレクトリに配置。

## 注意事項

- シェルスクリプトは `set -euo pipefail` 必須
- 温度取得は Intel X540-T2 (ixgbe) 想定
- `/config/` 以下はVyOS再起動後も永続化
