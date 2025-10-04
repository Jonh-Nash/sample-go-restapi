# Account API

RESTful 設計原則に基づく **アカウント認証型 API サーバー**実装です。
サーバレスではなく **アプリケーションサーバ**として実装。外部 FW 不使用（標準 `net/http`）。

## Tech

- Go 1.25.1
- bcrypt (`golang.org/x/crypto/bcrypt`)
- DDD レイヤ: `internal/domain` / `internal/usecase` / `internal/infrastructure/repository` / `internal/entrypoint/rest`

## ローカル環境

### 手元の Go 環境を利用する場合

```bash
make dev
```

`make dev` は API サーバーをバックグラウンドで起動し、`cmd/seed` のスクリプトを介してテストユーザを API 経由で投入します。
サーバーのみを起動したい場合は `make serve`、既存のサーバーに対してシードだけ行いたい場合は `make seed` を利用してください。

`PORT` 環境変数でリッスンポートを指定できます。未設定の場合は `8080` を利用します。

### Docker を利用する場合

起動

```bash
docker build -t account-api:local .
docker run --rm -p 8080:8080 account-api:local
```

seed

```bash
API_BASE_URL=http://localhost:8080 go run ./cmd/seed
```

## 公開

GitHub Actions で 自動デプロイ & 初期シード されます。

### 手元から Heroku にデプロイする場合

```bash
heroku login
heroku create # 最初のみ
git push heroku main # デプロイ
heroku logs --tail # ログ
```

```bash
API_BASE_URL=https://sample-go-restapi-de79e66392c3.herokuapp.com go run ./cmd/seed
```

Heroku ではアプリ起動時に `PORT` 環境変数が割り当てられるため、ローカルのような固定ポート指定は不要です。

Heroku で `web` dyno を 0 台にスケールしていると、アクセス時に `code=H14 "No web processes running"` が返ります。`heroku ps:scale web=1` を実行して dyno を起動してください。

## Kubernetes

Kustomize を利用しています。
docker image を release タグで作成してください。deployment で release タグで参照しているためです。
あとはよしなに、CI/CD ツール上などで Secret に必要なファイルをキーストアなどから取得して、マニフェストを apply してください。

### kind を使ってローカル環境で確認する場合

各種インストール

```bash
brew install kind
brew install kubectl
brew install kustomize
```

自己証明書を作成

```bash
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -sha256 -days 365 -nodes \
  -keyout ./kubernetes/overlays/local/secrets/tls.key -out ./kubernetes/overlays/local/secrets/tls.crt \
  -subj "/CN=account-api.example.com" \
  -addext "subjectAltName=DNS:account-api.example.com"
```

kind クラスタを作成

```bash
kind create cluster --name kdev --config ./kubernetes/kind/kind-ingress.yaml
kubectl get nodes # クラスタのノードを確認
```

クラスタに ingress-nginx をインストール

```bash
kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
```

リリースタグで docker image を作成して、kind クラスタに load

```bash
docker build -t account-api:release .
kind load docker-image account-api:release --name kdev
```

マニフェストを apply

```bash
kustomize build kubernetes/overlays/local > kubernetes/kind/dist.yaml
kubectl apply -f kubernetes/kind/dist.yaml
kubectl -n app get deploy,svc,ingress,secret
```

疎通確認

```bash
curl -k --resolve 'account-api.example.com:443:127.0.0.1' \
  https://account-api.example.com/healthz
```

クラスタを削除

```bash
kind delete cluster --name kdev
```

#### アプリケーションを更新する場合

```bash
docker build -t account-api:release .
kind load docker-image account-api:release --name kind
kubectl rollout restart deployment account-api -n app
```

## テスト

GitHub Actions で E2E テストが実行されます。

### E2E

```bash
brew install newman
newman run ./postman/accountapi-e2e.postman_collection.json
```
