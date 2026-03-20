# API Gateway Examples

Примеры использования API Gateway Forklore

## Authentication Examples

### 1. Register (Регистрация)

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice_wonder",
    "email": "alice@example.com",
    "password": "SecurePass123!@#"
  }'
```

Response:
```json
{
  "user": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "alice_wonder",
    "email": "alice@example.com",
    "role": "user",
    "status": "active",
    "created_at": "2026-03-14T00:00:00Z",
    "updated_at": "2026-03-14T00:00:00Z",
    "last_login_at": "2026-03-14T00:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "token_type": "Bearer",
    "access_expires_at": "2026-03-14T01:00:00Z",
    "refresh_expires_at": "2026-03-21T00:00:00Z",
    "session_id": "550e8400-e29b-41d4-a716-446655440001"
  }
}
```

### 2. Login (Вход)

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "login": "alice_wonder",
    "password": "SecurePass123!@#"
  }'
```

### 3. Refresh Token (Обновление токена)

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
  }'
```

### 4. Get Current User (Получить текущего пользователя)

```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

## Repository Examples

### Setup - Save Token

```bash
# После регистрации/входа, сохраните token:
TOKEN="eyJhbGciOiJIUzI1NiIs..."
USER_ID="550e8400-e29b-41d4-a716-446655440000"
```

### 1. Get Tags (Получить все теги)

```bash
curl -X GET http://localhost:8080/api/v1/repositories/tags \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "tags": [
    {
      "tag_id": "550e8400-e29b-41d4-a716-446655440010",
      "name": "Technology",
      "slug": "technology",
      "description": "Technology related content",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-14T00:00:00Z"
    },
    {
      "tag_id": "550e8400-e29b-41d4-a716-446655440011",
      "name": "Tutorial",
      "slug": "tutorial",
      "description": "Tutorial and guide content",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-14T00:00:00Z"
    }
  ]
}
```

### 2. Create Repository (Создать репозиторий)

```bash
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Go API Development",
    "slug": "go-api-development",
    "description": "A comprehensive guide to building APIs in Go",
    "tag_id": "550e8400-e29b-41d4-a716-446655440010",
    "visibility": "public",
    "type": "article"
  }'
```

Response:
```json
{
  "repository": {
    "repo_id": "660e8400-e29b-41d4-a716-446655440020",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "Go API Development",
    "slug": "go-api-development",
    "description": "A comprehensive guide to building APIs in Go",
    "visibility": "public",
    "type": "article",
    "tag": {
      "tag_id": "550e8400-e29b-41d4-a716-446655440010",
      "name": "Technology",
      "slug": "technology",
      "description": "Technology related content",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-03-14T00:00:00Z"
    },
    "parent_repo_id": null,
    "created_at": "2026-03-14T00:00:00Z",
    "updated_at": "2026-03-14T00:00:00Z",
    "deleted_at": null
  }
}
```

### 3. Get My Repositories (Получить мои репозитории)

```bash
curl -X GET "http://localhost:8080/api/v1/repositories/me?limit=10&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "repositories": [
    {
      "repo_id": "660e8400-e29b-41d4-a716-446655440020",
      "owner_id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Go API Development",
      "slug": "go-api-development",
      "description": "A comprehensive guide to building APIs in Go",
      "visibility": "public",
      "type": "article",
      "tag": { ... },
      "parent_repo_id": null,
      "created_at": "2026-03-14T00:00:00Z",
      "updated_at": "2026-03-14T00:00:00Z",
      "deleted_at": null
    }
  ],
  "total": 1
}
```

### 4. Get Repository by ID

```bash
curl -X GET http://localhost:8080/api/v1/repositories/660e8400-e29b-41d4-a716-446655440020 \
  -H "Authorization: Bearer $TOKEN"
```

### 5. Get Repository by Owner and Slug

```bash
curl -X GET http://localhost:8080/api/v1/users/$USER_ID/repositories/go-api-development \
  -H "Authorization: Bearer $TOKEN"
```

### 6. Update Repository (Обновить репозиторий)

```bash
curl -X PATCH http://localhost:8080/api/v1/repositories/660e8400-e29b-41d4-a716-446655440020 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Advanced Go API Development",
    "description": "A comprehensive guide to building advanced APIs in Go",
    "visibility": "private"
  }'
```

### 7. Fork Repository (Форкировать репозиторий)

Отметим, что форкировать можно только публичные репозитории других пользователей.

```bash
# Сначала получим ID публичного репозитория
REPO_TO_FORK="660e8400-e29b-41d4-a716-446655440020"

curl -X POST http://localhost:8080/api/v1/repositories/$REPO_TO_FORK/fork \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Fork of Go API Development",
    "slug": "my-fork-go-api",
    "description": "My personal fork with modifications"
  }'
```

Response:
```json
{
  "repository": {
    "repo_id": "770e8400-e29b-41d4-a716-446655440030",
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "My Fork of Go API Development",
    "slug": "my-fork-go-api",
    "description": "My personal fork with modifications",
    "visibility": "private",
    "type": "article",
    "tag": { ... },
    "parent_repo_id": "660e8400-e29b-41d4-a716-446655440020",
    "created_at": "2026-03-14T00:00:00Z",
    "updated_at": "2026-03-14T00:00:00Z",
    "deleted_at": null
  }
}
```

### 8. List User Repositories (Получить репозитории пользователя)

```bash
OTHER_USER_ID="550e8400-e29b-41d4-a716-446655440001"

curl -X GET "http://localhost:8080/api/v1/users/$OTHER_USER_ID/repositories?limit=5&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

Примечание: Будут видны только публичные репозитории, или если это текущий пользователь, то все.

### 9. List Forks (Получить форки репозитория)

```bash
REPO_ID="660e8400-e29b-41d4-a716-446655440020"

curl -X GET "http://localhost:8080/api/v1/repositories/$REPO_ID/forks?limit=10&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "repositories": [
    {
      "repo_id": "770e8400-e29b-41d4-a716-446655440030",
      "owner_id": "550e8400-e29b-41d4-a716-446655440001",
      "name": "Fork by Bob",
      "slug": "fork-by-bob",
      "description": "Bob's fork",
      "visibility": "public",
      "type": "article",
      "tag": { ... },
      "parent_repo_id": "660e8400-e29b-41d4-a716-446655440020",
      "created_at": "2026-03-14T12:00:00Z",
      "updated_at": "2026-03-14T12:00:00Z",
      "deleted_at": null
    }
  ],
  "total": 1
}
```

### 10. Delete Repository (Удалить репозиторий)

```bash
curl -X DELETE http://localhost:8080/api/v1/repositories/660e8400-e29b-41d4-a716-446655440020 \
  -H "Authorization: Bearer $TOKEN"
```

Возвращает 204 No Content при успехе.

## Error Examples

### 400 Bad Request

```bash
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AB"  # Too short
  }'
```

Response:
```json
{
  "code": 400,
  "message": "Key: 'CreateRepositoryRequest.Name' Error:Field validation for 'Name' failed on the 'min' tag"
}
```

### 401 Unauthorized

```bash
curl -X GET http://localhost:8080/api/v1/repositories/me
```

Response:
```json
{
  "code": 401,
  "message": "unauthorized"
}
```

### 403 Forbidden

```bash
# Пытаемся обновить чужой репозиторий
curl -X PATCH http://localhost:8080/api/v1/repositories/660e8400-e29b-41d4-a716-446655440020 \
  -H "Authorization: Bearer $OTHER_USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Hacked"}'
```

Response:
```json
{
  "code": 403,
  "message": "permission denied"
}
```

### 404 Not Found

```bash
curl -X GET http://localhost:8080/api/v1/repositories/00000000-0000-0000-0000-000000000000 \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "code": 404,
  "message": "not found"
}
```

### 409 Conflict

```bash
# Slug уже занят
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Go API Development",
    "slug": "go-api-development",  # Already exists
    "tag_id": "550e8400-e29b-41d4-a716-446655440010",
    "visibility": "public",
    "type": "article"
  }'
```

Response:
```json
{
  "code": 409,
  "message": "already exists"
}
```

## Using Postman

1. Импортируйте Swagger JSON:
   - `File` → `Import` → `Link`
   - Вставьте: `http://localhost:8080/swagger/swagger.json`

2. Добавьте переменные окружения:
   - `base_url` = `http://localhost:8080`
   - `token` = ваш JWT token
   - `repo_id` = ID репозитория

3. Используйте `{{base_url}}`, `{{token}}`, `{{repo_id}}` в ваших запросах

## Using Swagger UI

Откройте `http://localhost:8080/swagger/index.html` в браузере для интерактивной документации.
