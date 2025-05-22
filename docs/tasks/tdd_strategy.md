# テスト駆動開発 (TDD) 戦略ドキュメント

## 1. はじめに

### テスト駆動開発 (TDD) とは
テスト駆動開発（Test-Driven Development, TDD）は、ソフトウェア開発手法の一つであり、プログラムの実装よりも先にテストケースを記述することを特徴とする。このアプローチにより、開発者は要件をより深く理解し、設計の初期段階で問題を特定できるようになる。

### TDDのメリット
TDDを導入することで、以下のような多くのメリットが期待できる。
*   **品質向上**: テストケースを先に書くことで、コードの各部分が期待通りに動作することを保証しやすくなる。バグの早期発見にも繋がる。
*   **設計改善**: テスト容易性を考慮することで、自然と疎結合で凝集度の高い、より良い設計のコードが生まれやすくなる。
*   **リファクタリングの容易化**: 包括的なテストスイートが存在することで、安心してコードの改善（リファクタリング）を行うことができる。変更による意図しない副作用（デグレード）を検出しやすくなる。
*   **ドキュメントとしての役割**: テストコードは、そのコードがどのように動作すべきかを示す生きたドキュメントとしての役割も果たす。

## 2. TDDの基本サイクル

TDDは、以下の3つのステップを繰り返す短いサイクルで開発を進める。

*   **Red**: まず、新機能や修正点を検証するための「失敗する」テストコードを記述する。この時点では、対応する実装コードはまだ存在しないか、不完全であるためテストは失敗する（Red）。
*   **Green**: 次に、記述したテストが「成功する」ための最小限の実装コードを書く。ここでは、複雑なロジックや最適化は後回しにし、まずはテストをパスさせること（Green）を目標とする。
*   **Refactor**: 最後に、テストが成功する状態を維持したまま、実装コードの設計を改善したり、冗長な部分を排除したりするリファクタリングを行う。テストがあるため、リファクタリングによるバグの混入を恐れずに行える。

この「Red → Green → Refactor」のサイクルを繰り返すことで、堅牢で保守性の高いソフトウェアを段階的に構築していく。

## 3. 本プロジェクトにおけるTDDの実践

このプロジェクトにおいてTDDを効果的に実践するための具体的な指針を以下に示す。

### テストの種類と対象

*   **ユニットテスト**:
    *   **対象**: 個々の関数、メソッド、小さなコンポーネント。
    *   **目的**: 各ユニットが独立して正しく機能することを確認する。
    *   **例**: [`chat/prompt.go`](chat/prompt.go:0) のようなユーティリティ関数、[`history/history.go`](history/history.go:0) の履歴管理ロジックなど。
*   **インテグレーションテスト**:
    *   **対象**: 複数のモジュールやサービスが連携する部分。
    *   **目的**: モジュール間のインターフェースやデータフローが正しく動作することを確認する。
    *   **例**: [`chat/chat.go`](chat/chat.go:0) が [`history/history.go`](history/history.go:0) や外部APIクライアントと連携する部分。
*   **E2E (End-to-End) テスト (将来的な展望)**:
    *   **対象**: Discord Botとしてのユーザー操作から応答までの一連のフロー。
    *   **目的**: システム全体がユーザーの視点で期待通りに動作することを確認する。
    *   **備考**: 実装にはテスト用のDiscord Botアカウントや、メッセージ送受信をシミュレートする仕組みが必要になる場合がある。

### テストケースの記述

*   **Go標準の `testing` パッケージの利用**: Goの標準ライブラリである `testing` パッケージを基本として使用する。追加のテストフレームワークの導入は、必要性が明確になった時点で検討する。
*   **テーブル駆動テストの活用**: 複数の入力と期待される出力のパターンを効率的にテストするために、テーブル駆動テスト (Table-Driven Tests) を積極的に採用する。
    ```go
    func TestAdd(t *testing.T) {
        cases := []struct {
            name     string
            a, b     int
            expected int
        }{
            {"positive numbers", 2, 3, 5},
            {"negative numbers", -2, -3, -5},
            {"zero", 0, 0, 0},
        }
        for _, tc := range cases {
            t.Run(tc.name, func(t *testing.T) {
                actual := Add(tc.a, tc.b)
                if actual != tc.expected {
                    t.Errorf("Add(%d, %d) = %d; expected %d", tc.a, tc.b, actual, tc.expected)
                }
            })
        }
    }
    ```
*   **テストケースの命名規則**:
    *   テスト関数名: `Test` で始まり、テスト対象の関数名やメソッド名を続ける (例: `TestGetResponse`, `TestBuildFullInput`)。
    *   テーブル駆動テストのケース名 (`t.Run` の第一引数): テストケースの内容が具体的にわかるような名前を付ける (例: "正常系_メッセージのみ", "異常系_APIエラー")。

### モックとスタブ

外部API（Gemini API, Weather APIなど）やデータベースなど、テスト実行時に制御が難しい、あるいは実行コストが高い依存関係を持つコンポーネントのテストには、モックやスタブを活用する。

*   **インターフェースの活用**: 外部依存を持つコンポーネントはインターフェースを介して利用するように設計する。これにより、テスト時にはインターフェースを満たすモックオブジェクトに差し替えることが容易になる。
    ```go
    // 例: WeatherService のインターフェース
    type WeatherFetcher interface {
        Fetch(location string) (string, error)
    }

    // テスト対象の構造体
    type MyService struct {
        weather WeatherFetcher
    }

    // テスト用のモック
    type MockWeatherFetcher struct {
        MockFetch func(location string) (string, error)
    }

    func (m *MockWeatherFetcher) Fetch(location string) (string, error) {
        return m.MockFetch(location)
    }
    ```
*   **依存性の注入 (Dependency Injection)**: 構造体の初期化時などに、インターフェース型のフィールドに具体的な実装（本番用またはモック）を注入する。
*   **モックライブラリの検討**: 手動でのモック実装が煩雑になる場合は、Goで利用可能なモックライブラリ（例: `gomock`, `testify/mock`）の導入を検討する。ただし、導入の際は学習コストやプロジェクトへの適合性を考慮する。

### テストの実行

*   **全テストの実行**: プロジェクトルートで `go test ./...` コマンドを実行することで、全てのパッケージのテストを一度に実行できる。
*   **カバレッジ計測**: テストがコードのどの程度をカバーしているかを示すカバレッジを計測するには、`go test -coverprofile=coverage.out ./...` を実行し、`go tool cover -html=coverage.out` で結果をHTML形式で確認できる。目標カバレッジを設定することも有効だが、カバレッジの数値だけを追い求めるのではなく、重要なロジックが適切にテストされているかを重視する。

## 4. 具体的なテストコードの例

### 単純な関数のユニットテスト例
(上記テーブル駆動テストの `TestAdd` を参照)

### 外部依存を持つ関数のモックを使ったユニットテスト例

```go
// chat/chat.go の GetResponse が外部APIに依存すると仮定した場合のテストイメージ
// (実際には genai.GenerativeModel などをモックする必要がある)

type MockGenAIModel struct {
    GenerateContentFunc func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error)
}

func (m *MockGenAIModel) GenerateContent(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
    if m.GenerateContentFunc != nil {
        return m.GenerateContentFunc(ctx, parts...)
    }
    return nil, fmt.Errorf("GenerateContentFunc not implemented")
}

func TestChat_GetResponse_Success(t *testing.T) {
    mockModel := &MockGenAIModel{
        GenerateContentFunc: func(ctx context.Context, parts ...genai.Part) (*genai.GenerateContentResponse, error) {
            // 期待されるAPIレスポンスを模倣
            return &genai.GenerateContentResponse{
                Candidates: []*genai.Candidate{
                    {
                        Content: &genai.Content{
                            Parts: []genai.Part{genai.Text("Mocked LLM Response")},
                        },
                    },
                },
            }, nil
        },
    }
    
    // historyMgr やその他の依存も適切にモックまたは初期化する
    chatService := &Chat{ // 実際には NewChat 経由で初期化し、モックを注入する
        genaiModel: mockModel, 
        // ... other dependencies
    }

    resp, _, _, err := chatService.GetResponse("user1", "thread1", "testuser", "hello", "timestamp", "prompt")
    if err != nil {
        t.Fatalf("GetResponse failed: %v", err)
    }
    if resp != "Mocked LLM Response" {
        t.Errorf("Expected response 'Mocked LLM Response', got '%s'", resp)
    }
}
```
*注意: 上記はあくまで概念的な例であり、実際の `genai.GenerativeModel` のインターフェースや `Chat` 構造体の詳細に合わせて調整が必要となる。*

## 5. TDD推進のためのTips

*   **小さく始める**: 最初から全てのコードにTDDを適用しようとせず、まずは新しい機能やバグ修正など、限定的な範囲からTDDを試してみる。
*   **テストしやすい設計を心がける**: 関数やメソッドが単一責任の原則に従い、副作用が少ない純粋関数に近い形であるほど、テストは書きやすくなる。
*   **定期的なリファクタリング**: Greenのフェーズでテストをパスしたコードも、定期的に見直し、よりクリーンで効率的なコードへと改善していく。テストがあることで、このリファクタリングが安全に行える。
*   **テストコードもコードである**: テストコードも本番コードと同様に、可読性や保守性を意識して記述する。複雑すぎるテストは、それ自体がバグの原因となったり、メンテナンスの負担になったりする。

## 6. 今後の展望

*   **CI (Continuous Integration) へのテスト組み込み**: GitHub ActionsなどのCIサービスを利用して、コードがリポジトリにプッシュされるたびに自動的にテストを実行する仕組みを構築する。これにより、変更が既存の機能に悪影響を与えていないかを常に確認できるようになる。

---
このドキュメントが、本プロジェクトにおけるTDDの実践の一助となれば幸いである。
