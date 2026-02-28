# AGENTS.md for go-graft

このファイルは `github.com/Warashi/go-graft` リポジトリ固有の前提だけを記載する。
ユーザー単位の共通設定や他プロジェクト向けルールはここに書かない。

## プロジェクト概要

- **未公開のため、変更プランを立てる時は後方互換性を破壊する前提で考えること。**
- `go-graft` は `go test -overlay` を使う Go のミューテーションテストフレームワーク。
- 提供形態はライブラリのみ（CLI は提供しない）。
- 公開APIはルートパッケージ（`Engine`, `Register`, `RegisterMethodCallSwap`, `RegisterFunctionCallSwap`, `Context`, `Report`）。

## 実装上の前提（現状）

- ミュータントは「1ミュータント = 1 mutation point（1ノード差し替え）」を前提に組み立てる。
- 実行ステータスは `Killed` / `Survived` / `Unsupported` / `Errored` を分けて扱う。
- テスト実行は `internal/runner` で行い、`go test` に `-overlay`, `-failfast`, `-parallel=1` を付ける。
- `internal/testdiscover` は通常テストを抽出しつつ、`(*graft.Engine).Run` に到達可能な mutation test を自動除外し、`//gograft:include` / `//gograft:exclude` のディレクティブで上書きできる。
- 主要処理は `internal/projectload` -> `internal/testdiscover` -> `internal/mutationpoint` -> `internal/mutantbuild` -> `internal/testselect` -> `internal/runner` -> `internal/reporting` の責務分割で構成される。

## 開発時の標準チェック

- lint: `go vet ./...`
- test: `go test ./...`

## AGENTS.md のメンテナンス要件

次の変更を行ったら、この `AGENTS.md` も同じ作業で更新する。

- 主要ディレクトリの責務や処理フローを変えたとき
- ライブラリの既定挙動（主要設定値や実行フロー）を変えたとき
- 実行ステータス区分や runner の実行方式を変えたとき
- 標準チェックコマンド（lint/test）を変えたとき

更新時は「このリポジトリに常時適用される事実」だけを残し、一時的な運用メモは書かない。
