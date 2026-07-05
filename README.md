## Misskeyイベント管理ツールAPI

MisskeyのMiAuth認証を利用したスケジュール調整ツールのAPIです。
イベントの作成や参加者の管理、期日通知をMisskeyアカウントで行えます。

## 必須環境

- Go 1.26以上
- PostgreSQL 16
- Docker（推奨）

### 本番環境

`.env.example`を`.env`にコピーして値を設定し、Docker Composeで起動します。

cp .env.example .env
docker compose up -d

### 開発環境

開発用の`.env.dev`には既に値が設定されています。Docker Composeを使わず手動で起動する場合は以下です。

cp .env.dev .env
go run ./cmd/server

Docker Composeを使う場合は以下で起動します。

docker compose -f docker-compose.dev.yml up -d

### 設定

環境変数の詳細は`.env.example`を参照してください。
