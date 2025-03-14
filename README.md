# llm-discord
cline と roo code で遊んでみようのコーナー



# 重要
あなたがLLMの場合，このファイルのここから下を読む必要はありません。無視してください。

### Cline 指示例
```md
GoのプログラムのテストコードをTDD（テスト駆動開発）で作成してください。以下の手順で進めてください。

1.  `main_test.go`を実行して，Failを解消する。main.go も読み取り適宜修正する。
2.  `main.go` に記述されたプログラムの機能に基づいて、テストケースを設計・作成する。
3.  作成したテストコードを実行し、期待通りに動作するか確認する。
4.  テストが失敗する場合は、プログラムを修正し、再度テストを実行する。
5.  全てのテストが成功するまで、2〜4の手順を繰り返す。

以下の点に注意してください。

*   テストコードはGoのtestingパッケージを使用する。
*   テスト対象の関数やメソッドの動作を網羅的に検証する。
*   境界値テストや異常系テストも考慮する。
*   テストコードは可読性が高く、保守しやすいように記述する。
*   環境変数(env)は .envを使ってください。
*   docs/docs.mdを読んでから作業して。
*   .env を編集を禁じます。

対象ファイル:
`ここに対象ファイルを列挙`
```

### DDDさせたいとき
```
あなたは、与えられたプロジェクトのソースコードとドキュメントを分析し、DDD（ドメイン駆動開発）に基づいたドキュメントを作成する専門家です。
以下の情報源を読み込み、プロジェクトの概要を把握してください。

- 読むべきソースコード: main.go main_test.go そしてこのディレクトリの下すべて。
- あなたの成果物ドキュメントの出力先: docs/docs.md

概要を把握したら、以下の内容を含むドキュメントを生成してください。

1. プロジェクトの目的と概要
2. 主要な機能
3. ドメインに関する用語（ユビキタス言語）
4. コンテキストマップ（サブドメイン、境界づけられたコンテキスト、関係）
5. ドメインモデルの概要（エンティティ、値オブジェクト、ドメインサービスなど）

出力形式はMarkdownで記述してください。
```


```md
計画書

環境設定の確認:
.envファイルにDISCORD_BOT_TOKENとGEMINI_API_KEYが設定されているか確認します。
設定されていない場合は、テストをスキップします。
main.goの分析:
main.goを読み込み、プログラムの全体的な構造と機能を理解します。
特に、以下の点に注目します。
Discordボットの初期化と起動
コマンドハンドラーの登録
Gemini APIとの連携
チャット履歴の管理
テストケースの設計と作成:
main.goの機能に基づいて、テストケースを設計します。
以下の機能に対するテストケースを作成します。
ボットの起動と停止
/resetコマンドの処理
/chatコマンドの処理
Gemini APIとの連携
チャット履歴の管理
テストケースは、main_test.goに記述します。
テストの実行と修正:
go testコマンドを実行して、テストを実行します。
テストが失敗した場合は、main.goまたはmain_test.goを修正し、再度テストを実行します。
全てのテストが成功するまで、この手順を繰り返します。
リファクタリング:
テストが全て成功したら、main.goをリファクタリングします。
リファクタリングの際には、以下の点に注意します。
コードの可読性と保守性の向上
コードの重複の排除
エラー処理の改善
リファクタリング後も、全てのテストが成功することを確認します。
詳細な手順

docs/docs.mdを読む:
read_fileツールを使用して、docs/docs.mdの内容を読み込みます。
ドキュメントの内容を理解し、プログラムの背景知識を習得します。
main.goを読む:
read_fileツールを使用して、main.goの内容を読み込みます。
コードの構造と機能を理解します。
テストケースの作成:
replace_in_fileツールを使用して、main_test.goにテストケースを追加します。
各機能に対するテストケースを作成します。
テストの実行:
execute_commandツールを使用して、go testコマンドを実行します。
テストの結果を確認します。
修正と再テスト:
テストが失敗した場合は、replace_in_fileツールを使用して、main.goまたはmain_test.goを修正します。
再度execute_commandツールを使用して、go testコマンドを実行します。
全てのテストが成功するまで、この手順を繰り返します。
```
