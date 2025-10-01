# Account API (DDD / Go / net/http / SQLite / bcrypt)

RESTful 設計原則に基づく **アカウント認証型 API サーバー**実装です。  
サーバレスではなく **アプリケーションサーバ**として実装。外部 FW 不使用（標準 `net/http`）。

## Tech

- Go 1.25.1
- SQLite (`modernc.org/sqlite`) _CGO 不要_
- bcrypt (`golang.org/x/crypto/bcrypt`)
- DDD レイヤ: `internal/domain` / `internal/usecase` / `internal/infrastructure/repository` / `internal/entrypoint/httpapi`

## Run (local)

```bash
go mod tidy
PORT=8080 DB_DSN='file:/data/users.db?cache=shared&_busy_timeout=5000' SEED_TEST_USER=true \
go run ./cmd/api-server
```

### Health

```bash
curl -s http://localhost:8080/healthz
# => {"message":"ok"}
```

## Docker

```bash
docker build -t account-api:local .
docker run --rm -p 8080:8080 -e SEED_TEST_USER=true \
  -v $(pwd)/data:/data account-api:local
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

---

## API 仕様（重要な文言は課題文と**完全一致**）

### 1) POST /signup

**Request**

```bash
curl -s -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"TaroYamada","password":"PaSSwd4TY"}'
```

**Success (200)**

```json
{
  "message": "Account successfully created",
  "user": { "user_id": "TaroYamada", "nickname": "TaroYamada" }
}
```

**Error examples (400)**

- 必須欠落:

```bash
curl -s -X POST http://localhost:8080/signup -H 'Content-Type: application/json' -d '{}'
# {"message":"Account creation failed","cause":"Required user_id and password"}
```

- 長さ不正 / 文字種不正 / 重複（各 cause は課題文と**完全一致**）:

```
{"message":"Account creation failed","cause":"Input length is incorrect"}
{"message":"Account creation failed","cause":"Incorrect character pattern"}
{"message":"Account creation failed","cause":"Already same user_id is used"}
```

### 2) GET /users/{user_id} （Basic 認証必須）

```bash
curl -s -u TaroYamada:PaSSwd4TY http://localhost:8080/users/TaroYamada
```

**Success (200, 設定済みの例)**

```json
{
  "message": "User details by user_id",
  "user": {
    "user_id": "TaroYamada",
    "nickname": "たろー",
    "comment": "僕は元気です"
  }
}
```

**未設定例（comment 無し / nickname は user_id と同値）**

```json
{
  "message": "User details by user_id",
  "user": { "user_id": "TaroYamada", "nickname": "TaroYamada" }
}
```

**Error**

- 認証失敗: `401 {"message":"Authentication failed"}`
- 未存在: `404 {"message":"No user found"}`
- （仕様準拠）path と認証ユーザが不一致でも `401`

### 3) PATCH /users/{user_id} （Basic 認証必須）

```bash
curl -s -X PATCH -u TaroYamada:PaSSwd4TY http://localhost:8080/users/TaroYamada \
  -H 'Content-Type: application/json' \
  -d '{"nickname":"たろー","comment":"僕は元気です"}'
```

**Success (200)**

```json
{
  "message": "User successfully updated",
  "user": {
    "user_id": "TaroYamada",
    "nickname": "たろー",
    "comment": "僕は元気です"
  }
}
```

**Error**

- 両方未指定: `400 {"message":"User updation failed","cause":"Required nickname or comment"}`
- 制約/制御文字: `400 {"message":"User updation failed","cause":"String length limit exceeded or containing invalid characters"}`
- user_id/password を含めた: `400 {"message":"User updation failed","cause":"Not updatable user_id and password"}`
- 認証失敗: `401 {"message":"Authentication failed"}`
- 権限なし（path ≠ 認証ユーザ）: `403 {"message":"No permission for update"}`
- 未存在: `404 {"message":"No user found"}`

### 4) POST /close （Basic 認証必須）

```bash
curl -s -X POST -u TaroYamada:PaSSwd4TY http://localhost:8080/close
```

**Success (200)**

```json
{ "message": "Account and user successfully removed" }
```

**Auth error (未存在も 401)**

```json
{ "message": "Authentication failed" }
```

---

```

---

### 動作上のポイント

- **CORS**: `Access-Control-Allow-Origin: *` / `GET,POST,PATCH,OPTIONS`
- **WWW-Authenticate**: 401 応答時に `Basic realm="account-api"` を付与
- **Body 制限**: 1MiB (`http.MaxBytesReader`)
- **Timeout**: Read/Write/Idle を設定
- **DB**: `file:/data/users.db?cache=shared&_busy_timeout=5000`（デフォルト）
- **Seed**: `SEED_TEST_USER=true` の場合のみ初期ユーザを作成（**"Test～"** は作成しません）
- **DELETE**: `/close` は**物理削除**
- **文字列制約**: 課題文に厳密準拠（長さ/パターン/制御コード）

---

## 提出物について

- 本回答の各ファイルをそのまま配置してビルド可能です。
- 可能であれば一式を zip 化して **`account-api-ddd-repo-go.zip`** として提出してください。
- `env.yml` には公開 URL（ngrok 等）を設定してください（サンプルは `env.yml.example` 参照）。

---

必要に応じて、このまま貼り付けてご利用ください。ビルド・起動・テスト方法は `README.md` のとおりです。
```
