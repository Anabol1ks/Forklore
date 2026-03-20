# Forklore API Gateway

API Gateway для микросервисной архитектуры Forklore, предоставляющий единую точку входа для всех операций с репозиториями и аутентификацией.

## Архитектура

```
┌──────────────┐
│  Client      │
└──────┬───────┘
       │ HTTP
       ▼
┌──────────────────────┐
│   API Gateway        │
│  (этот сервис)       │
└──────┬───────┬───────┘
       │ gRPC  │ gRPC
       ▼       ▼
    ┌─────┐ ┌──────────────┐
    │Auth │ │  Repository  │
    │Svc  │ │    Svc       │
    └─────┘ └──────────────┘
```

## Features

- ✅ Управление репозиториями (CRUD операции)
- ✅ Аутентификация и авторизация
- ✅ Форк репозиториев
- ✅ Полная Swagger документация
- ✅ Пагинация для списков
- ✅ Обработка ошибок gRPC → HTTP

## Быстрый старт

### Prerequisites

- Go 1.25+
- Docker & Docker Compose (опционально)
- swag CLI для генерации Swagger: `go install github.com/swaggo/swag/cmd/swag@latest`

### Environment Variables

```bash
# Gateway
GATEWAY_PORT=8080
ENV=development

# Microservices
AUTH_SERVICE_ADDR=localhost:8081
REPOSITORY_SERVICE_ADDR=localhost:8082
```

### Запуск

```bash
# Установить зависимости
go mod tidy

# Построить
go build -o gateway cmd/gateway/main.go

# Запустить
./gateway
```

## API Endpoints

### Структура API

```
/api/v1/
├── /auth
│   ├── POST   /register          - Регистрация
│   ├── POST   /login             - Вход
│   ├── POST   /refresh           - Обновление токенов
│   ├── POST   /logout            - Выход из сессии
│   ├── POST   /logout-all        - Выход из всех сессий (protected)
│   └── GET    /me                - Текущий пользователь (protected)
├── /repositories
│   ├── POST   /                  - Создание (protected)
│   ├── GET    /me                - Мои репозитории (protected)
│   ├── GET    /tags              - Все теги (protected)
│   ├── GET    /:repo_id          - По ID (protected)
│   ├── PATCH  /:repo_id          - Обновление (protected)
│   ├── DELETE /:repo_id          - Удаление (protected)
│   ├── POST   /:repo_id/fork     - Форк (protected)
│   └── GET    /:repo_id/forks    - Форки репозитория (protected)
└── /users
    └── /:owner_id  
        └── /repositories
            ├── GET   [/]         - Репозитории пользователя (protected)
            └── GET   /:slug      - По владельцу и slug (protected)
```

### Авторизация / Authentication

#### Register
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securePassword123"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "login": "john_doe",
  "password": "securePassword123"
}
```

#### Refresh Tokens
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Get Current User
```http
GET /api/v1/auth/me
Authorization: Bearer <access_token>
```

### Repositories / Репозитории

#### Create Repository
```http
POST /api/v1/repositories
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "My Article",
  "slug": "my-article",
  "description": "An interesting article",
  "tag_id": "550e8400-e29b-41d4-a716-446655440000",
  "visibility": "public",
  "type": "article"
}
```

#### Get Repository by ID
```http
GET /api/v1/repositories/{repo_id}
Authorization: Bearer <access_token>
```

#### Get Repository by Slug
```http
GET /api/v1/users/{owner_id}/repositories/{slug}
Authorization: Bearer <access_token>
```

#### Update Repository
```http
PATCH /api/v1/repositories/{repo_id}
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Updated Article",
  "visibility": "private",
  "description": "Updated description"
}
```

#### Delete Repository
```http
DELETE /api/v1/repositories/{repo_id}
Authorization: Bearer <access_token>
```

#### Fork Repository
```http
POST /api/v1/repositories/{repo_id}/fork
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Forked Article",
  "slug": "forked-article",
  "description": "My fork"
}
```

#### List My Repositories
```http
GET /api/v1/repositories/me?limit=10&offset=0
Authorization: Bearer <access_token>
```

#### List User Repositories
```http
GET /api/v1/users/{owner_id}/repositories?limit=10&offset=0
Authorization: Bearer <access_token>
```

#### List Repository Forks
```http
GET /api/v1/repositories/{repo_id}/forks?limit=10&offset=0
Authorization: Bearer <access_token>
```

#### List Repository Tags
```http
GET /api/v1/repositories/tags
Authorization: Bearer <access_token>
```

## Repository Types

- **article** - статья или пост
- **notes** - заметки или конспект
- **mixed** - смешанный тип контента

## Visibility

- **public** - доступна всем
- **private** - видна только владельцу

## HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200  | Success |
| 201  | Created |
| 204  | No Content |
| 400  | Bad Request |
| 401  | Unauthorized |
| 403  | Forbidden |
| 404  | Not Found |
| 409  | Conflict (e.g., duplicate slug) |
| 500  | Internal Server Error |
| 503  | Service Unavailable |

## Error Response Format

```json
{
  "code": 400,
  "message": "invalid request"
}
```

## gRPC Error to HTTP Mapping

| gRPC Code | HTTP Code |
|-----------|-----------|
| InvalidArgument | 400 |
| Unauthenticated | 401 |
| PermissionDenied | 403 |
| NotFound | 404 |
| AlreadyExists | 409 |
| Unavailable | 503 |
| DeadlineExceeded | 504 |

## Swagger Documentation

Полная интерактивная документация API доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

## Разработка

### Регенерация Swagger Docs

После изменения комментариев в обработчиках:

```bash
make swag
# или
swag init -g cmd/gateway/main.go
```

### Project Structure

```
api-gateway/
├── cmd/
│   └── gateway/
│       └── main.go              # Entry point
├── config/
│   └── config.go                # Configuration loading
├── internal/
│   ├── clients/
│   │   ├── auth_client.go      # gRPC client for auth-service
│   │   └── repository_client.go # gRPC client for repository-service
│   ├── handlers/
│   │   ├── auth_handler.go      # HTTP handlers for auth
│   │   ├── repository_handler.go # HTTP handlers for repositories
│   │   ├── mapper.go            # gRPC response mapping
│   │   └── response.go          # Error handling
│   ├── middleware/
│   │   ├── auth.go              # Authorization middleware
│   │   └── logging.go           # Request logging
│   ├── models/
│   │   ├── auth.go              # Auth models
│   │   └── repository.go        # Repository models
│   └── router/
│       └── router.go            # Route definitions
├── docs/
│   ├── swagger.json
│   ├── swagger.yaml
│   └── docs.go
├── go.mod
├── go.sum
├── makefile
└── README.md
```

### Middleware

#### AuthRequired

Проверяет наличие валидного JWT токена в заголовке Authorization.

```go
protected.Use(middleware.AuthRequired())
```

#### Logger

Логирует все входящие HTTP запросы с использованием zap.

## Миграция с других API Gateway

### Совместимость

API Gateway полностью совместим с существующей архитектурой Forklore и использует gRPC для comunicкации с микросервисами.

### Добавление новых микросервисов

1. Создайте gRPC client в `internal/clients/`
2. Создайте HTTP handler в `internal/handlers/`
3. Добавьте модели в `internal/models/`
4. Обновите `config.go` для новой переменной окружения
5. Инициализируйте в `main.go`
6. Добавьте routes в `router.go`
7. Регенерируйте Swagger docs

## Troubleshooting

### Gateway не может подключиться к microservices

1. Проверьте что microservices запущены
2. Проверьте переменные окружения:
   ```bash
   echo $AUTH_SERVICE_ADDR
   echo $REPOSITORY_SERVICE_ADDR
   ```
3. Проверьте сетевую connectivity:
   ```bash
   nc -zv localhost 8081
   nc -zv localhost 8082
   ```

### Swagger UI не загружается

1. Убедитесь что документация была сгенерирована:
   ```bash
   make swag
   ```
2. Проверьте что `docs.go` существует в `docs/` папке

### 401 Unauthorized

1. Убедитесь что token передан в заголовке Authorization
2. Формат: `Authorization: Bearer <token>`
3. Проверьте что token ещё не истёк

## Production Build

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gateway cmd/gateway/main.go
```

## License

MIT
