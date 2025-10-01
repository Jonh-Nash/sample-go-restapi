# Account API (DDD / Go / net/http / SQLite / bcrypt)

RESTful 設計原則に基づく **アカウント認証型 API サーバー**実装です。
サーバレスではなく **アプリケーションサーバ**として実装。外部 FW 不使用（標準 `net/http`）。

## Tech

- Go 1.25.1
- bcrypt (`golang.org/x/crypto/bcrypt`)
- DDD レイヤ: `internal/domain` / `internal/usecase` / `internal/infrastructure/repository` / `internal/entrypoint/rest`

## Run (local)

```bash
make dev
```

`make dev` は API サーバーをバックグラウンドで起動し、`cmd/seed` のスクリプトを介してテストユーザを API 経由で投入します。サーバーのみを起動したい場合は `make serve`、既存のサーバーに対してシードだけ行いたい場合は `make seed` を利用してください。

### Health

```bash
curl -s http://localhost:8080/healthz
# => {"message":"ok"}
```

## Docker

```bash
docker build -t account-api:local .
docker run --rm -p 8080:8080 account-api:local

# 別ターミナル
API_BASE_URL=http://localhost:8080 go run ./cmd/seed
```

## Kubernetes

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
# ClusterIP の 80 -> 8080
```

## 公開（例: ngrok）

ローカルで起動後:

```bash
ngrok http http://localhost:8080
# 表示された https URL を env.yml の API_BASE_URL に記入
```
