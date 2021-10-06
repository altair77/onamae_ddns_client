# Onamae.com DDNS Client

お名前.comののLinux用DDNSクライアント

# 使い方

## Golang

1. `config.yml`を準備
1. 以下のコマンドを実行
   ```sh
   $ go mod download
   $ go run main.go
   ```

## Docker

1. `config.yml`を準備
1. 以下のコマンドを実行
   ```sh
   $ docker-compose up -d
   ```

# `config.yml`

`config.example.yml`をコピーして、`config.yml`を作成してください。

## auth

お名前.comにログインするための、お名前IDとパスワードを設定します。  
`お名前ID:パスワード`の文字列をBase64にエンコードして設定してください。

## domains.name

更新するドメイン名を設定します。

## domains.hosts.name

更新するサブドメインを設定します。  
空文字を設定することでサブドメインなしを設定します。

# 動作

起動時にグローバルIPアドレスを取得し、お名前.comへ更新をリクエストします。  
その後、10分おきにグローバルIPアドレスを取得します。その際にIPアドレスに変更があればお名前.comに更新をリクエストします。
