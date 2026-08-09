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

## フロントエンド配信（OGP付き）

Goサーバーはビルド済みのフロントエンド(`frontend/dist`)を`static/frontend_dist`へ
コピーして起動すると、SPA本体と `https://<host>/events/{id}` のOGP（イベント名を
`og:title` にしたHTML）を同オリジンで配信します。

ビルド前に以下を実行してください（フロントエンドリポジトリ側）。

```sh
# frontend リポジトリ
npm run build
cp -r dist/. ../backend/static/frontend_dist/
```

Dockerビルドも含め、`static/frontend_dist` に `index.html` があればそのまま組み込まれます
（無い場合はプレースホルダのみが組み込まれます）。

### Netlify + バックエンドの2台構成

フロントをNetlifyに置いたままOGPも出したい場合は、NetlifyのEdge Function
(`frontend/netlify/edge-functions/ogp.ts`) が `/events/{id}` をバックエンドの
`GET /public/events/{id}` へプロキシします。このルートはOGP注入済みのSPA HTMLを返します。

- Netlifyサイトに環境変数 `BACKEND_PUBLIC_URL`（例 `https://api.example.com`）を設定
- バックエンドは `og:url` 組み立て時に Edge Function が付与する `X-Forwarded-Host`/`X-Forwarded-Proto` を尊重します
