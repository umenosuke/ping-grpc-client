# icmp ping をリモートで撃ってもらうクライアント

## これは何

[これ](https://github.com/umenosuke/ping-grpc-server)のクライアント

## Demo

![demo](https://raw.github.com/wiki/umenosuke/ping-grpc-client/images/Demo.gif)

## 使い方

### 例

#### 対話モードで実行

```
#TLSを利用しない場合
./ping-grpc-client -S "(サーバーの)`IP`:`port`" -noUseTLS
```

```
#TLSを利用する場合
#  CA証明書 "./ca.crt"
#  クライアント証明書 "./client_pinger.crt"
#  クライアント秘密鍵 "./client_pinger.pem"
#を用意してください
./ping-grpc-client -S "(サーバーの)`IP`:`port`"
```

[コマンドの中身の説明](https://github.com/umenosuke/ping-grpc-client/blob/master/README_command.md)

#### サブコマンドで実行

実行コマンドの末尾にサブコマンドを追記することで非対話モードで動作します

```
./ping-grpc-client [引数] {サブコマンド} [サブコマンドの引数]
```

サブコマンドは下記の通りです

通常の引数でエラー(file not found 等)があった場合、サブコマンドは実行されないです(help 含め)

```
$ ./ping-grpc-client help
[help]
start "{target list path}"                 : start pinger
start "{target list path}" "{description}" : start pinger
stop "{pingerID}"                          : stop pinger

list       : show pinger list summary
list long  : show pinger list verbose
list short : show pinger id list

info "{pingerID}"   : show pinger info
result "{pingerID}" : show ping result
count "{pingerID}"  : show ping statistics

help : (this) show help
```

基本的に各サブコマンドの出力などは対話モードと同様です

### TLS を利用する場合

[ここ](https://github.com/umenosuke/x509helper)などを参考に

- CA の証明書
- クライアント証明書と秘密鍵

を作成してください

### オプションなど

```
$ ./ping-grpc-client -help
Usage of ./ping-grpc-client:
  -S string
        server address:port (shorthand) (default "127.0.0.1:5555")
  -cCert string
        client certificate file path (default "./client_pinger.crt")
  -cKey string
        client private key file path (default "./client_pinger.pem")
  -caCert string
        CA certificate file path (default "./ca.crt")
  -config string
        config json string (default "{}")
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
  -s string
        server address:port (shorthand) (default "127.0.0.1:5555")
  -server string
        server address:port (default "127.0.0.1:5555")
  -v    show version (shorthand)
  -version
        show version
```

### コンフィグの内容について

ping の開始リクエストで利用します

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
- Docker とか

### コマンド

クローン

```
git clone --recursive git@github.com:umenosuke/ping-grpc-client.git
cd ping-grpc-client
```

設定読み込み

```
source .script/_conf.sh
```

ビルド用のコンテナを立ち上げ

```
docker-compose -f .docker/docker-compose.yml up -d
```

linux&amd64 用バイナリを作成(ビルドターゲットは任意で変更してください)<br>

```
docker exec -it go_build_${_PRJ_NAME} target_data/.script/go_build.sh 'linux' 'amd64' './src' "build/${_PRJ_NAME}"
```

ビルド用のコンテナをお片付け

```
docker-compose -f .docker/docker-compose.yml down
```

バイナリはこれ

```
build/ping-grpc-client
```
