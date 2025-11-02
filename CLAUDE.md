# CLAUDE.md - ETC明細Goモジュール プロジェクトコンチE�E��E�スチE

## プロジェクト概要E
ETC明細チE�E�EタをWebスクレイピングで取得し、データベ�Eスに保存するGoモジュール、E
db-handler-serverパターンに従ったハンドラー実裁E�E��E�の移行中、E

## 技術スタチE�E��E�
- **言誁E*: Go 1.21+
- **フレームワーク**: gRPC + grpc-gateway (chi から移衁E
- **Protocol Buffers**: API定義とコード生戁E
- **チE�E�Eタベ�Eス**: db_service (Fiber実裁E via gRPC
- **通信**: gRPC (server_repo統吁E
- **スクレイピング**: Playwright-go
- **チE�E��E�チE�E��E�ング**: testify/mock, table-driven tests (100%カバレチE�E��E�目樁E
- **依存管琁E*: Go Modules, buf (Protocol Buffers)
- **アーキチE�E��E�チャ**: etc_meisai ↁEgRPC ↁEdb_service (Fiber)

## プロジェクト構造
```
etc_meisai/
├── src/
│   ├── proto/           # Protocol Buffers定義
│   ├── pb/              # 生成されたgRPCコード
│   ├── grpc/            # gRPCサーバー実装
│   ├── registry/        # サービス登録（desktop-server統合用）
│   ├── services/        # ビジネスロジック層
│   ├── scraper/         # Webスクレイピング機能
│   └── models/          # データモデル
├── handlers/            # HTTPハンドラー（レガシー）
├── tests/               # テストファイル
├── main.go              # スタンドアロンサーバーエントリーポイント
└── downloads/           # CSVファイル保存先
```

## 主要機�E
1. **ETC明細ダウンローチE*: 褁E�E��E�アカウント対応、E�E��E�同期処琁E
2. **チE�E�Eタ処琁E*: CSV解析、データ変換、バルク保孁E
3. **マッピング管琁E*: ETC明細とチE�E��E�タコチE�E�Eタの関連付け�E�E�E�Etc_num活用�E�E�E�E
4. **進捗追跡**: リアルタイム進捗通知�E�E�E�ESE対応！E
5. **自動�EチE�E��E�ング**: dtako_row_idとの精寁E�E�EチE�E��E�ング

## 最近の変更 (v0.0.23 - セッションフォルダ共有)
- **複数アカウント対応**: 1回の実行で複数アカウントのCSVを同一フォルダに保存
  - ScraperConfigにSessionFolderフィールドを追加
  - ProcessAsync()で共通のタイムスタンプ付きセッションフォルダを生成
  - ファイル名にアカウント名プレフィックスを自動付与
- **ファイル管理の改善**: downloads/YYYYMMDD_HHMMSS/配下に整理
  - 例: ohishiexp_202510182104.csv, ohishiexp1_202510182105.csv

## 変更履歴 (v0.0.22 - CSV明細ダウンロード機能修正)
- **ダイアログ自動承認**: CSV確認ダイアログを自動的にOK
  - Initialize時にグローバルダイアログハンドラーを設定
  - playwright.Dialogの正しい型キャストとAccept()呼び出し
- **ダウンロードハンドラー安定化**: SaveAs()のハング問題を解決
  - goroutineとタイムアウトを使用してダウンロードを監視
  - ファイル保存完了を正しく検出
- **CSV取得フロー簡素化**: 不要なチェックボックス選択処理を削除
- **詳細ログ出力**: ダイアログ検出、ダウンロードイベント、ファイル保存の進捗を表示

## 変更履歴 (v0.0.21 - ログイン処理修正)
- **ログインフロー修正**: 実際のETCサイト構造に合わせて修正
  - ログインリンクをクリックして専用ログインページに遷移
  - 正しいフォームフィールド名 (risLoginId/risPassword) を使用
  - 正しいログインボタンセレクタを使用
- **コード簡素化**: 冗長なデバッグコードと複数セレクタ試行ロジックを削除
- **テスト更新**: 新しいログインフローに対応、全テストパス (100%)

## 変更履歴 (v0.0.20 - desktop-server統合対応)
- **Registry パッケージ追加**: desktop-server統合用のサービス登録機能
- **スタンドアロンサーバー改善**: main.goをgRPCサーバーとして実行可能に
  - デフォルトポート: 50052（desktop-server統合用）
  - ヘルプコマンド追加（--help）
  - 別プロセス実行を推奨
- **README更新**: スタンドアロン実行方法とdesktop-server統合ガイド追加
- **統合アーキテクチャ確立**: 別プロセス + gRPC通信方式

## desktop-serverとの統合（推奨アプローチ）

### 別プロセス方式
```
desktop-server.exe ← gRPC Client → etc_meisai_scraper.exe (別プロセス)
```

**メリット:**
- ✅ desktop-serverのバイナリサイズが小さいまま
- ✅ 環境依存性の分離（Playwright依存）
- ✅ スクレイピング処理がデスクトップアプリに影響しない
- ✅ 必要な時だけ起動可能（オンデマンド起動）

詳細は `C:\go\desktop-server\docs\etc_meisai_scraper_integration_spec.md` を参照

## スコープ外�E機�E
- Excel/PDF エクスポ�Eト機�E
- 統計情報生�E機�E
- キャチE�E��E�ュ機�E�E�E�E�ユーザー要求により除外！E

## パフォーマンス目樁E
- CSVファイル1丁E�E��E�めE秒以冁E�E��E�処琁E
- メモリ使用釁E00MB以丁E
- 同時ダウンローチEアカウントまで
- チE�E��E�ト実行時閁E0秒以冁E�E��E��E�EチE�E��E�トスイート！E
- チE�E��E�トカバレチE�E��E�100%維持E

## 統合アーキチE�E��E�チャ (db_service via gRPC)
```
etc_meisai:
  HTTP API (handlers/) ↁEService Layer (services/) ↁEgRPC Client
                                                           ↁE
                                                     [gRPC Protocol]
                                                           ↁE
db_service (Fiber):
  gRPC Server ↁERepository (GORM) ↁEMySQL Database
```

### 統合コンポ�EネンチE
- **統吁Eepository**: `src/repositories/etc_integrated_repo.go`
- **gRPCクライアンチE*: `src/clients/db_service_client.go`
- **互換性レイヤー**: `src/adapters/etc_compat_adapter.go`
- **統合テスチE*: `tests/contract/`, `tests/integration/`
- **チE��トファイルの配置**: `tests/`チE��レクトリのみ�E�Esrc/`には配置しなぁE- 憲法原剁E��E

### 統合仕槁E
- **チE�E�EタモチE�E��E�**: [data-model.md](specs/001-db-service-integration/data-model.md)
- **API契紁E*: [contracts/](specs/001-db-service-integration/contracts/)
- **開発ガイチE*: [quickstart.md](specs/001-db-service-integration/quickstart.md)

## 環境変数
- `ETC_CORP_ACCOUNTS`: アカウント情報（カンマ区切り）
  - 形式: `userid1:password1,userid2:password2`
  - 法人・個人の区別なく全アカウントを指定
- `ETC_HEADLESS`: Headlessモード（デフォルト: true）
  - `true` または未設定: Headlessモード（ブラウザ非表示）
  - `false`: ブラウザ表示モード（デバッグ用）

**レガシー環境変数**（後方互換性のため残存）:
- `ETC_CORPORATE_ACCOUNTS`: 法人アカウント（カンマ区切り）
- `ETC_PERSONAL_ACCOUNTS`: 個人アカウント（カンマ区切り）
- 注: 現在のコードでは法人・個人の区別なく同じ処理を行うため、`ETC_CORP_ACCOUNTS`への統一を推奨

## チE�E��E�トコマンチE
```bash
go test ./...                    # 単体テスチE
go test ./tests/integration -v   # 統合テスチE
```

## ビルド！E�E��E�衁E

### 開発ビルド
```bash
go build -o etc_meisai
./etc_meisai
```

### リリースビルド (Windows)

Windowsバイナリのリリースビルドは、専用のビルドスクリプトを使用します。

#### PowerShell版（推奨）
```powershell
# 最新のgit tagから自動的にバージョンを取得してビルド
.\build-release.ps1

# 特定のバージョンを指定してビルド
.\build-release.ps1 -Version "v0.0.25"
```

#### Bash版（Git Bash/WSL）
```bash
# 最新のgit tagから自動的にバージョンを取得してビルド
./build-release.sh

# 特定のバージョンを指定してビルド
./build-release.sh v0.0.25
```

**ビルド出力:**
- バイナリ: `etc_meisai_scraper-{VERSION}-windows-amd64.exe`
- アーカイブ: `etc_meisai_scraper-{VERSION}-windows-amd64.zip`
  - バイナリ + README.md + CLAUDE.md を含む

**ビルドオプション:**
- 最適化ビルド (`-ldflags "-s -w"`) でバイナリサイズを削減
- バージョン情報を埋め込み (`-X main.Version=...`)
- CGO無効 (`CGO_ENABLED=0`) でポータブルなバイナリを生成

---
*最終更新: 2025-10-18 | v0.0.23*

# Hook出力�E琁E�E持E��

## カバレチE��惁E��が届いた場吁E
hookから「📁E[Hook] Coverage analysis:」などのカバレチE��惁E��を受信したら！E
- **忁E��ユーザーに表示すること**
- パ�EセンチE�Eジとパッケージ名を整形して表示
- 80%未満の低カバレチE��を強調表示
- 表示例！E
  ```
  📊 カバレチE��レポ�Eト！E
  - src/models: 85.2% ✁E
  - src/services: 72.5% ⚠�E�E(改喁E��忁E��E
  - src/repositories: 90.1% ✁E
  ```

## フォーマットエラーが届いた場吁E
「⚠�E�EFORMAT ERROR DETECTED:」などのフォーマットエラーを受信したら！E
- **ユーザーに通知**
- 具体的な問題箁E��を表示
- 即座に修正を提桁E

## go vetエラーが届いた場吁E
go vetエラーを受信したら！E
- **エラーを�E確に表示**
- エラーの意味を説昁E
- 修正方法を提侁E

## Constitution違反が届いた場吁E
Constitution違反�E�例：src/にチE��トファイル�E�を受信したら！E
- **即座にユーザーに警呁E*
- 憲法違反�E琁E��を説昁E
- 正しい場所への移動を提桁E
