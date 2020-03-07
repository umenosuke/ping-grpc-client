# icmp ping をリモートで撃ってもらうクライアント

## これは何

[これ](https://github.com/umenosuke/ping-grpc-server)のクライアント
- pingを撃つデーモン
- ping対象の指定などはgRPCクライアントから
- 一回のリクエスト内の対象へはpingを並列で撃つ

ping結果を見る箇所と撃ち箇所を別にできるので<br>
PCからのpingが遮断されているときの疎通確認や<br>
AS内部と外部からの疎通確認とかに

## Demo

![demo](https://raw.github.com/wiki/umenosuke/ping-grpc-client/images/Demo.gif)

## 使い方

### 例

実行
```
#TLSを利用する場合
#  CA証明書 "./ca.crt"
#  クライアント証明書 "./client_pinger.crt"
#  クライアント秘密鍵 "./client_pinger.pem"
#を用意してください
./ping-grpc-client -S "(サーバーの)`IP`:`port`"

#TLSを利用しない場合
./ping-grpc-client -S "(サーバーの)`IP`:`port`" -noUseTLS
```

### TLS利用する場合

[ここ](https://github.com/umenosuke/x509helper)などを参考に

- CAの証明書
- クライアント証明書と秘密鍵

を作成してください

### オプションなど

```
$ ./ping-grpc-client -help
Usage of ./ping-grpc-client:
  -S string
        server address:port (default "127.0.0.1:5555")
  -cCert string
        client certificate file path (default "./client_pinger.crt")
  -cKey string
        client private key file path (default "./client_pinger.pem")
  -caCert string
        CA certificate file path (default "./ca.crt")
  -configPath string
        config file path
  -debug
        print debug log
  -noColor
        disable colorful output
  -noUseTLS
        disable tls
  -printConfig
        show default config
  -version
        show version
```

### コンフィグの内容について

pingの開始リクエストで利用します

[ここ](https://github.com/umenosuke/ping-grpc-client/blob/master/src/config.go)の
```
type Config struct
```
がそのままエンコードされた形です<br>
値の詳細についてはコメントを参照してください

コンフィグファイルに無い項目はデフォルト値になります

## ビルド方法

### ビルドに必要なもの

- git
- Dockerとか

### コマンド

クローン
```
git clone --recursive git@github.com:umenosuke/ping-grpc-client.git
cd ping-grpc-client
```

ビルド用のコンテナを立ち上げ
```
_USER="$(id -u):$(id -g)" docker-compose -f .docker/docker-compose.yml up -d
```

protoのコンパイル
```
docker exec -it proto_build_ping-grpc-client target_data/.script/proto_build.sh
```

linux&amd64用バイナリを作成(ビルドターゲットは任意で変更してください)<br>
```
docker exec -it go_build_ping-grpc-client target_data/ping-grpc-client/.script/go_build.sh 'linux' 'amd64' 'build/ping-grpc-client'
```

ビルド用のコンテナをお片付け
```
_USER="$(id -u):$(id -g)" docker-compose -f .docker/docker-compose.yml down
```

バイナリはこれ

```
build/ping-grpc-client
```
