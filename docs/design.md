# Go Mutation Testing Framework Design Document

> 対象：Go 1.18+
> 目的：このドキュメントだけを渡して、実装者（ジュニア）が開発を進められる粒度まで具体化する
> スコープ：フレームワーク（ライブラリ）中心。CLI は薄く載せる

---

## 0. サマリ

本プロジェクトは、Go の **`go test -overlay`** を軸に、元ソースを破壊せずにミュータント（変異コード）を生成・並列実行する **ミューテーションテスト・フレームワーク**を作る。

設計の核は以下：

* **安全な並列**：1ミュータントごとに **temp dir + overlay JSON** を作り、`go test -overlay` をプロセス単位で並列実行
* **型安全なルールAPI**：Generics で `ast.Node` 型を固定し、`*Context`（独自）を渡して **型情報/祖先文脈**を使える
* **ASTはCopy-on-Write**：変更経路（Path）のみシャローコピーして差し替え、全コピーを避ける（反射は避け、`go generate` で静的生成）
* **テスト選択戦略**：FN（実行漏れ）を致命傷扱いしつつ、GoのパッケージDAG依存を使って FP（余計な実行）を刈り込むハイブリッド
* **保証は範囲を限定**：万能なFNゼロ保証はしない。**保証対象外**を明示し、怪しい結果を「Survived」と誤報しない

---

## 1. ゴール / 非ゴール

### 1.1 ゴール（必須）

1. ユーザーが型安全にミューテーションルールを登録できる（Generics、型アサーション不要）
2. ミュータント生成は安全（元AST破壊なし）かつ高速（CoW）
3. ミュータントを並列実行できる（worker pool）
4. `go test` は **パッケージ単位で起動**し、`-run` で **関数単位に絞る**
5. `-failfast` を利用し、ミュータントが kill されたら即終了できる
6. タイムアウトは **Killed** として扱う
7. どの結果も説明可能（どのルール、どの箇所、どのテストで kill されたか）

### 1.2 非ゴール（v1ではやらない）

* 変異ファイルの内容ハッシュによる永続キャッシュ（I/O削減の高度最適化）
* 反射/unsafe/cgo を含むあらゆるケースでの FN ゼロ保証
* 1箇所から複数ミュータント候補を返す API（v1では「高々1」）
* テストの外部資源（固定ポート/DB等）を自動で安全化する高度な隔離

---

## 2. 設計上の“契約”（暗黙知の明文化）

### 2.1 不変条件（強い制約）

* **1ミュータントで同時に書き換える `ast.Node` は1つまで**
* **1ルール×1ノード → 高々1ミュータント**

  * 複数バリエーションが必要ならルールを複数登録する
* `go test` は **パッケージ単位**、`-run` で **テスト関数単位**
* `go test -parallel` は常に **1**
* 並列数ノブは **ワーカー数（同時 `go test` プロセス数）**のみ
* 隔離は **TMPDIR まで**（GOCACHE/GOPATHは go command が適切に扱う前提）

### 2.2 保証範囲（C：保証対象を限定）

* “ツールが約束できる範囲”をドキュメントで明示し、それ以外は **Unsupported** として結果を分離する
* Unsupported を **Survived と誤報しない**（信頼を守る）

---

## 3. 用語

* **Mutation Point**：変異対象となるASTノード（1ミュータント=1ポイント）
* **Mutant**：1つの変異（1ノード差し替え）を適用した実行単位
* **Rule**：ユーザーが登録する変異ルール
* **Overlay**：`go test -overlay` で差し替えるファイルマッピングJSON
* **Killed**：テスト失敗/タイムアウト/実行エラーにより変異が検知された
* **Survived**：選択されたテストが全て成功し、変異が検知されなかった
* **Unsupported**：保証範囲外/解析不能などで信頼できない（Survived扱いにしない）

---

## 4. 公開API（フレームワーク）

> ※標準 `context.Context` と混同しないよう、独自の文脈は `*mutest.Context` と呼ぶ

### 4.1 Engine と Register

```go
package mutest

import "go/ast"

type Engine struct {
    Config Config

    // internal fields:
    // rules registry, analyzers, logger, etc.
}

// ユーザーが変異ルールを登録する（Generics）
func Register[T ast.Node](e *Engine, mutate func(c *Context, n T) (T, bool), opts ...RuleOption)
```

#### Rule の契約

* `mutate` は **高々1回の変異**を返す
* `bool` が false の場合、そのノードからはミュータントを生成しない
* 引数 `n` はエンジン側で **シャローコピーされたノード**（元AST破壊を防ぐ）
* ルールは **このノード以外を変更しない**（祖先/兄弟/別ファイルを書き換えない）
* 複数候補が欲しい場合は、同じ型で別ルールを複数 Register する

#### RuleOption（最低限）

ジュニア実装でも扱えるように必須オプションは絞る。

* `WithName(name string)`：レポートに出すためのルール名（未指定なら `rule#N`）
* `WithDeepCopy()`：子孫まで書き換える等、深い変形が必要な場合に使用（v1では実装コスト次第で後回し可）

```go
type RuleOption func(*ruleConfig)
```

### 4.2 Engine 実行API

```go
type Config struct {
    Workers int           // 同時 go test プロセス数（唯一の並列ノブ）
    MutantTimeout time.Duration
    BaseTempDir string    // 空なら os.MkdirTemp のデフォルト
    KeepTemp bool         // デバッグ用に temp を残す
}

func (e *Engine) Run(runCtx context.Context, patterns ...string) (*Report, error)
```

* `patterns` は `go/packages` に渡すパッケージパターン（例：`./...`）
* `runCtx` は全体のキャンセル/タイムアウト制御用

---

## 5. 内部アーキテクチャ（モジュール構成）

### 5.1 主要コンポーネント

1. **Loader**

   * `go/packages` でモジュール全体（Tests含む）をロード
   * `types.Info` を保持（パース時に1回だけ型チェック）
2. **MutationPoint Finder**

   * AST を走査し、登録済みルールの対象型に一致するノードを拾う
   * （推奨）カバレッジで対象を絞る
3. **Analyzer**

   * テスト関数一覧の抽出
   * callgraph（CHA/RTA）
   * 副作用データフロー（pkg var / receiver field）
   * パッケージ依存DAGの reverse deps
4. **Mutant Builder**

   * ルール適用 → 変異ノード生成
   * CoW で `*ast.File` を組み立て
   * overlay 用の変異ファイル出力（temp）
5. **Runner**

   * worker pool
   * `go test -overlay` をパッケージ単位で順次実行（-runで関数絞り、-failfast）
6. **Reporter**

   * 結果集計・表示（Killed/Survived/Unsupported を明確に分ける）

---

## 6. 詳細設計

## 6.1 パッケージロード（go/packages）

### 目的

* AST / Types / TypesInfo / Import graph / Tests を一度に取得する

### 実装指針

`packages.Load` を以下の Mode で呼ぶ：

* `NeedName`
* `NeedFiles`
* `NeedCompiledGoFiles`（overlayのキーに使う絶対パスが欲しい）
* `NeedSyntax`
* `NeedTypes`
* `NeedTypesInfo`
* `NeedImports`
* `NeedDeps`

`Tests: true` を指定してテストパッケージ（`p_test`等）もロードする。

---

## 6.2 テスト関数の抽出

### 対象

* トップレベルの `func TestXxx(t *testing.T)`
  （サブテスト `t.Run` はトップレベルが走ればOK、v1では個別選択しない）

### 実装

各 `packages.Package` の `Syntax` を走査して `*ast.FuncDecl` を抽出する。

判定：

* 名前が `Test` + 先頭大文字（`TestFoo`）
* 引数が1つで `*testing.T`

  * 最初は AST で形を見る（`*ast.StarExpr` + `*ast.SelectorExpr`）でも可
  * 可能なら `types.Info` でより正確に判定

保持する構造：

```go
type TestRef struct {
    PkgID string       // packages.Package.ID
    ImportPath string  // go test に渡すパス
    Name string        // TestFoo
}
```

---

## 6.3 Mutation Point の抽出

### 6.3.1 方針

* 登録済み Rule の型 `T` にマッチするノードのみ対象
* 可能ならカバレッジで対象を絞る（実行時間の爆発を避ける）

### 6.3.2 AST走査

* `ast.Inspect` で DFS
* 祖先スタック `path []ast.Node` を維持
* 同時に「現在の enclosing function」も維持（テスト選択の seed に必要）

保持する構造：

```go
type MutationPoint struct {
    PkgID string
    PkgImportPath string
    File *ast.File
    FilePath string           // compiled go file の絶対パス
    Node ast.Node             // オリジナルノード
    Path []ast.Node           // File -> ... -> Node
    Pos token.Position
    EnclosingFunc *ast.FuncDecl // ない場合（トップレベルなど）は nil
}
```

### 6.3.3 1ミュータント=1ノード契約の反映

* ミュータント生成時は「このポイントの Node」だけが変わる設計に固定する

---

## 6.4 `*Context`（独自文脈）

```go
type Context struct {
    Fset  *token.FileSet
    Pkg   *packages.Package
    File  *ast.File
    Types *types.Info

    Path []ast.Node

    cloneMap map[ast.Node]ast.Node // clone -> original
}
```

提供メソッド（最低限）：

* `TypeOf(node ast.Node) types.Type`
* `Original(node ast.Node) ast.Node`（cloneMap逆引き）
* （任意）`Parent() ast.Node` / `Ancestor(n int)` 等のヘルパ

> 重要：`Context` は **mutate コールバック中のみ有効**。保持/共有はしない。

---

## 6.5 CoW（Copy-on-Write）AST生成

### 目的

* 元ASTの破壊を防ぐ（ポインタ汚染防止）
* ただし全ASTコピーは遅いので、変更経路（Path）のみコピーする

### アルゴリズム（必須）

入力：

* `pathOrig []ast.Node`（File -> ... -> target）
* `nodeOrig`（target）
* `nodeMut`（ルールが返した変異ノード）

処理（上に向かって差し替え）：

1. `childClone = nodeMut`、`cloneMap[childClone]=nodeOrig`
2. 親へ向かって順に：

   * `parentClone := shallowCopy(parentOrig)`
   * `cloneMap[parentClone]=parentOrig`
   * `replaceChild(parentClone, childOrig, childClone)`

     * ここで **sliceフィールドはコピーして差し替える**（CoWの肝）
3. 最上位 `*ast.File` を得る

### 反射禁止（性能要件）

* `shallowCopy` / `replaceChild` は `go generate` で `go/ast` 定義から **静的生成**する
* 生成物は `internal/astcow/generated_*.go` 等に置く

最低限必要な生成関数：

* `func shallowCopy(n ast.Node) ast.Node`
* `func replaceChild(parent ast.Node, oldChild ast.Node, newChild ast.Node) bool`

---

## 6.6 ミュータントの出力と Overlay JSON

### 6.6.1 temp dir レイアウト（推奨）

`<tmp>/mutest-<id>/`

* `overlay.json`
* `overlay/`（変異ファイル置き場）
* `tmp/`（TMPDIR用）

### 6.6.2 変異ファイル生成

* `go/format` を使い `*ast.File` を整形して出力
* 原則 v1 は **変異対象の1ファイルのみ**を overlay に載せる

### 6.6.3 overlay.json 形式

`go test -overlay` が読む JSON：

```json
{
  "Replace": {
    "/abs/path/original.go": "/abs/path/mutant-temp/overlay/original.go"
  }
}
```

* キーは `packages.Package.CompiledGoFiles` の絶対パスを使う（ズレを避ける）

---

## 6.7 テスト選択（Hybrid）

### 目的

* 実行時間を抑えつつ、FN（実行漏れ）を極小化したい
* ただし万能保証はしない（保証範囲外は Unsupported）

### 6.7.1 事前計算（1回だけ）

1. **パッケージ依存グラフ（DAG）**

   * `go/packages` の Imports から `pkg -> imported` を作る
   * reverse deps を作る（`depender -> depended` の逆）
2. **callgraph（CHA / RTA）**

   * SSAを構築し、callgraphを作る
   * reverse edges を作る（callee -> callers）
3. **副作用データフロー**

   * pkg-level var の read/write
   * receiver field の read/write
   * writer->reader の依存エッジを作る

### 6.7.2 ミュータントごとの選択

seed：変異点の enclosing function

1. reverse callgraph で到達可能なテストを集める
2. 副作用エッジも含めて到達可能性を拡張（保守的に）
3. **パッケージDAGの reverse deps** で刈り込み

   * 「変異パッケージに依存しないパッケージのテスト」は **安全に捨てる**

### 6.7.3 結果のグルーピング（go test 実行単位）

* パッケージごとにテスト関数名を集め、`-run` の正規表現を作る：

`^(TestA|TestB|TestC)$`（各 name は QuoteMeta）

### 6.7.4 空集合の扱い（重要：誤報防止）

* 何らかの理由で `Tfinal` が空になった場合は **Unsupported** として扱う

  * **Survived にはしない**
  * レポートに理由（例：test selection produced 0 tests）を記載

---

## 6.8 実行（Runner）

### 6.8.1 実行コマンド（固定仕様）

ミュータントごとに、対象パッケージを順に実行する。

* `go test <pkg> -run <regex> -failfast -parallel=1 -count=1 -overlay=<overlay.json>`

> `-parallel=1` 固定（内側並列は常に殺す）
> `-p` は制御しない（外側ワーカー並列のみがユーザー制御ノブ）

### 6.8.2 ワーカー数（唯一の並列ノブ）

* `Config.Workers` が同時 `go test` プロセス数
* 各ミュータント内ではパッケージを **逐次**で実行（掛け算で過負荷になるのを避ける）

### 6.8.3 TMPDIR 隔離

* 各 `go test` 実行で `TMPDIR=<mutantTempDir>/tmp` を環境変数で指定
* GOCACHE / GOPATH はいじらない（go commandが適切に扱う前提）

### 6.8.4 タイムアウト

* ミュータント単位の `context.WithTimeout` を作り `exec.CommandContext` で実行
* タイムアウトは **Killed** 扱い

### 6.8.5 早期終了

* いずれかのパッケージで `go test` が non-zero を返したらその時点で **Killed** として打ち切り
* `-failfast` によりパッケージ内でも早期終了を期待

---

## 7. 結果モデルとレポート

### 7.1 ステータス

```go
type Status int
const (
    Killed Status = iota
    Survived
    Unsupported
    Errored
)
```

### 7.2 Report

* 集計：

  * total, killed, survived, unsupported
  * mutation score（killed/(killed+survived)、unsupported除外）
* 各ミュータントの詳細：

  * ID、RuleName、file:line:col、対象パッケージ
  * 実行したパッケージと `-run` 対象
  * Killed の場合：失敗コマンド、stdout/stderr、タイムアウト情報

---

## 8. 保証対象外（ドキュメントに明示する項目）

少なくとも以下を「保証対象外/サポート外」として明記する：

### 8.1 ルール実装のサポート外

* 1ミュータントで複数 `ast.Node` を同時に変更する
* `WithDeepCopy` なしで子孫ノードを直接書き換える
* 変異点とは無関係な祖先/兄弟ノードの変更
* 型情報の整合が崩れるような新規AST大量生成（cloneMapで追えない）

### 8.2 解析由来の保証対象外

* reflection / unsafe / cgo 等で callgraph・副作用解析が成立しないケース
* 動的ディスパッチや別名解析が必要な高度ケース（v1では保守的にしか扱えない）

### 8.3 テスト環境由来の保証対象外

* 固定ポート、共有DB、共有ファイルなどで並列に壊れるテスト

  * 緩和策：ワーカー数を下げる（最悪1）
* TMPDIR 以外の共有資源に依存するテスト

> 重要：保証対象外に触れた場合、結果は **Unsupported** に寄せ、Survived誤報を避ける

---

## 9. 実装手順（ジュニア向けマイルストーン）

### Milestone 1：Engine骨格と Rule登録

* `Engine`, `Config`, `Register`, `RuleOption`, `Run` の型を確定
* ルール registry を作る（型 T ごとにリスト）

### Milestone 2：go/packages ロード + テスト抽出

* `packages.Load(Tests:true)` を実装
* テスト関数一覧 `[]TestRef` を取れるようにする

### Milestone 3：MutationPoint 抽出

* AST traversal で `MutationPoint` を収集
* path stack と enclosing func の保持

### Milestone 4：CoW（まずは手書きでも可→後で生成へ）

* 最初は対象ノード種を限定して手書きで `shallowCopy/replaceChild` を作って動かす
* 動いたら `go generate` で全 `go/ast` 対応へ拡張

### Milestone 5：Overlay生成 + `go test -overlay` 単体実行

* 1ミュータントを temp に書き出し、手動で `go test -overlay` が動くことを確認
* `TMPDIR` の設定もここで確認

### Milestone 6：Runner（worker pool）実装

* `Workers` 個の goroutine
* mutant キューから取り出して `go test` を回す
* タイムアウト/Killed/Survived の判定

### Milestone 7：`-run` で関数単位に絞る + `-failfast` + `-parallel=1`

* パッケージごとに regex を作って実行
* まずは「全テスト」→次に「指定テスト」へ段階的に

### Milestone 8：テスト選択（Hybrid）導入

* まずは Phase 3 のみ（依存パッケージの全テスト）で cap を実装し、動作確認
* 次に callgraph（CHA）を入れる
* 最後に副作用エッジを入れる（爆発してもDAGでcapされる）

---

## 10. 参考：主要フロー疑似コード

### 10.1 Run 全体

```go
func (e *Engine) Run(runCtx context.Context, patterns ...string) (*Report, error) {
    proj := LoadProject(patterns)         // go/packages
    tests := DiscoverTests(proj)          // []TestRef
    graphs := PrecomputeGraphs(proj, tests) // deps, callgraph, side-effects

    points := FindMutationPoints(proj, e.rules)
    workCh := make(chan MutationPoint)
    resultCh := make(chan MutantResult)

    startWorkers(e.Config.Workers, workCh, resultCh, graphs, e.Config)

    for _, p := range points {
        select { case <-runCtx.Done(): break }
        workCh <- p
    }
    close(workCh)

    report := CollectResults(resultCh)
    return report, nil
}
```

### 10.2 MutationPoint → Mutant（1ノード×1ルール）

```go
for each rule applicable to point.Node:
    ctx := NewContext(point, proj.TypesInfo, path, cloneMap)
    nodeCopy := ShallowCopy(point.Node) // engine側
    mutated, ok := rule(ctx, nodeCopy)
    if !ok { continue }

    fileMut := CoWCloneFile(point.Path, point.Node, mutated, ctx.cloneMap)
    mutant := BuildOverlayAndMutant(fileMut, point, rule)
    mutant.TestsByPackage = SelectTests(mutant, graphs) // hybrid
    enqueue(mutant)
```

---

## 11. 実装上の注意（落とし穴）

* **sliceフィールドの差し替えは必ず slice コピー**（CoWの要）
* `packages.Package.CompiledGoFiles` の絶対パスを overlay のキーに使う（ズレ防止）
* `-run` regex は `regexp.QuoteMeta` を使う（安全）
* `go test` 出力は killed の解析に重要。stdout/stderr を保存する
* `KeepTemp` を用意して、失敗ミュータントを再現しやすくする

---

## 12. この設計で守るもの（最重要）

* **Survived 誤報で信頼を壊さない**
  → 保証外は Unsupported に逃がす
* **並列安全とデバッグ容易性**
  → 1mutant=1temp、-parallel=1、外側ワーカーのみ並列
* **ユーザーDX**
  → Generics + `*Context` + CoW の安全な AST 変形
