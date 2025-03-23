gemini だけではなく，ollamaを使う。

ollama のエンドポイント 192.168.XXX.XXX/ollama (リバースプロキシを使っているのでポート指定は不要)
エンドポイントはmodel.jsonに定義する。
ollama の項目が true の場合は ollamaを使う。通常は gemini を使う。

ollama.goを作成して，他のモジュールと繋ぐ。
