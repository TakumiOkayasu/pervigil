# CLAUDE.md

## プロジェクト概要

**Pervigil**: VyOSルーター向け監視ツール（Discord通知対応）。
- NIC温度監視・速度制限制御
- コスト監視（Anthropic Admin API）
- Discord Bot連携

## 技術スタック

| 項目 | 内容 |
|------|------|
| 言語 | Go 1.26 |
| 対象OS | VyOS (Linux) |
| 通知 | Discord Webhook / Bot |
| 外部API | Anthropic Admin API |
| ビルド | Docker (本番) / ローカルgo build (開発) |
| テスト | 標準 `testing` パッケージ（モックライブラリ禁止） |

## 開発コマンド

```bash
cd bot

# テスト
go test ./...                           # 全体
go test ./internal/monitor/             # パッケージ単位
go test ./internal/monitor/ -run TestX  # 単一関数

# ビルド (Docker = 本番)
docker build -f Dockerfile.build -t pervigil-builder .

# ビルド (ローカル = 開発)
CGO_ENABLED=0 go build -o pervigil-monitor ./cmd/pervigil-monitor
CGO_ENABLED=0 go build -o pervigil-bot ./cmd/pervigil-bot

# コード品質 (コミット前に必須・全エラーゼロを確認してから報告)
gofmt -w . && go vet ./... && staticcheck ./...
```

## アーキテクチャ

2バイナリ構成: `pervigil-monitor`（監視デーモン）と `pervigil-bot`（Discord Bot）。

```text
cmd/           ← エントリポイント。DI組立 + メインループのみ
  ↓ import
internal/
  monitor/     ← ビジネスロジック (NIC/ログ/コスト監視)
  handler/     ← Botコマンドハンドラ
  ↓ import
  temperature/ ← センサー抽象化 (ISP + DI)
  anthropic/   ← Anthropic Admin APIクライアント
  sysinfo/     ← システム情報取得
  notifier/    ← Discord Webhook通知
  config/      ← 設定読み込み (.env)
```

**レイヤールール（違反禁止）**:

| ルール | 内容 | 理由 |
|--------|------|------|
| `cmd/` は `internal/*` のみ import | エントリポイントの責務を限定 | 循環依存防止 |
| `monitor/`, `handler/` は下位パッケージのみ import | レイヤー順守 | 循環依存防止 |
| `temperature/`, `notifier/`, `config/` 等は相互 import 禁止 | 同一レイヤー横断禁止 | 依存方向の一方向化 |
| 外部 I/O は interface 経由で注入 | 直接呼び出し禁止 | テスト可能性・差し替え容易性 |

## 設計パターン

| パターン | 適用箇所 | 実装例 |
|---|---|---|
| ISP (Interface Segregation) | temperature/sensor.go | `commandRunnable`, `fileReadable`, `globbable` を分離 |
| DI (Dependency Injection) | 全パッケージ | `GetNICTempWith(iface, deps)` — 本番は `osDeps{}`, テストはモック |
| Functional Options | monitor/nic.go | `NewNICMonitor(WithNotifier(n), WithThresholds(t))` |
| FSM + ヒステリシス | monitor/nic.go | `Normal→Warning→Critical` 遷移、復旧は `Recovery(65℃)` で判定 |
| Sentinel Error | temperature/sensor.go | `ErrSensorUnavailable` — `errors.Is()` で判定 |

## テスト規約

- **モック**: interface を満たす struct をテストファイル内に定義。モックライブラリは使わない（禁止）
- **テーブル駆動**: 複数ケースは `tests := []struct{...}` + `for _, tt := range tests`
- **DI**: `XxxWith(deps)` 関数でテスト用依存を注入。公開API `Xxx()` は `osDeps{}` を渡すだけ

## 禁止事項

| 禁止操作 | 理由 |
|----------|------|
| モックライブラリの使用 | 標準 `testing` パッケージで十分、依存増加を防ぐ |
| 同一レイヤー間の直接 import | 循環依存・責務混在が起きる |
| 外部 I/O の直接呼び出し（interface 経由なし） | テスト不可能になる |
| `gofmt`/`go vet`/`staticcheck` 未実行でのコミット | 品質基準を守るため |
| `staticcheck` エラーを残したまま完了報告 | 未検証での報告は禁止 |

## 完了前チェックリスト

```
[ ] gofmt -w . でフォーマット済み
[ ] go vet ./... でエラーゼロ
[ ] staticcheck ./... でエラーゼロ
[ ] go test ./... が全部パス
[ ] レイヤールール違反なし（import確認）
```

## 環境変数

必須: `DISCORD_WEBHOOK_URL`（monitor）、`BOT_TOKEN`（bot）。
その他のオプション変数は [README.md](README.md) を参照。
`.env` は実行ファイルと同じディレクトリに配置。

## NIC温度閾値

| 温度 | 状態 | アクション |
|---|---|---|
| <70℃ | 正常 | — |
| 70–85℃ | 警告 | Discord通知 |
| >85℃ | 危険 | 速度1Gbps制限 |
| <65℃ | 復旧 | 速度制限解除 |

## 注意事項

- 温度取得は Intel X540-T2 (ixgbe) 想定
- VyOS の `/config/` 以下は再起動後も永続化
- デプロイ手順は [README.md](README.md) を参照
