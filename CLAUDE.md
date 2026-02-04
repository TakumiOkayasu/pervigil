# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

Pervigil: VyOSルーター向け監視ツール。Discord通知対応。

## アーキテクチャ

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

## 開発コマンド

### テスト

```bash
cd bot
go test ./...
```

### ビルド

```bash
cd bot
# 監視デーモン
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-monitor ./cmd/pervigil-monitor

# Discord Bot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o pervigil-bot ./cmd/pervigil-bot
```

### VyOSデプロイ

```bash
scp bot/pervigil-monitor vyos@<IP>:/config/scripts/pervigil/
scp bot/pervigil-bot vyos@<IP>:/config/scripts/pervigil/
```

## 環境変数 (pervigil-monitor)

| 変数 | 必須 | デフォルト | 用途 |
|------|------|-----------|------|
| DISCORD_WEBHOOK_URL | ✅ | - | Webhook URL |
| NIC_INTERFACE | - | eth1 | 監視NIC |
| CHECK_INTERVAL | - | 60 | チェック間隔(秒) |
| STATE_FILE | - | /tmp/pervigil-state | 状態ファイル |
| LOG_FILE | - | /var/log/syslog | 監視ログ |

## 環境変数 (pervigil-bot)

| 変数 | 必須 | 用途 |
|------|------|------|
| BOT_TOKEN | ✅ | Discord Bot Token |
| GUILD_ID | - | サーバーID (コマンド即時反映用) |

`.env` は実行ファイルと同じディレクトリに配置。

## NIC温度閾値

| 温度 | 状態 | アクション |
|------|------|------------|
| <70℃ | 正常 | - |
| 70-85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | 速度1Gbps制限 |
| <65℃ | 復旧 | 速度制限解除 |

## Go コード品質

**必須**: コミット前に以下を実行

```bash
cd bot
gofmt -w .              # フォーマット
go vet ./...            # 静的解析
staticcheck ./...       # 追加の静的解析
```

| ツール | 用途 | バージョン |
|--------|------|-----------|
| gofmt | フォーマット | Go標準 |
| go vet | 静的解析 | Go標準 |
| staticcheck | 追加lint | latest |

**staticcheck インストール**: `go install honnef.co/go/tools/cmd/staticcheck@latest`

## 注意事項

- 温度取得は Intel X540-T2 (ixgbe) 想定
- `/config/` 以下はVyOS再起動後も永続化
- Go: latest LTS を使用
