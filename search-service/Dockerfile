# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app/search-service

# Копируем go.mod/go.sum search-service и локальные модули, нужные для replace
COPY search-service/go.mod search-service/go.sum ./
COPY pkg/pb ../pkg/pb
COPY pkg/utils ../pkg/utils

# Загружаем зависимости
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Копируем исходный код search-service
COPY search-service/. .

# Собираем бинарники для всех компонентов
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/search-service ./cmd/service/main.go
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate/main.go

FROM alpine:latest

# Устанавливаем сертификаты для HTTPS, timezone data и dos2unix
RUN apk --no-cache add ca-certificates tzdata dos2unix

# Создаем непривилегированного пользователя
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Копируем бинарники из builder
COPY --from=builder /bin/search-service /app/search-service
COPY --from=builder /bin/migrate /app/migrate

# Копируем entrypoint скрипт и конвертируем окончания строк
COPY search-service/entrypoint.sh /app/entrypoint.sh
RUN dos2unix /app/entrypoint.sh && chmod +x /app/entrypoint.sh

# Меняем владельца файлов
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Экспонируем порт gRPC
EXPOSE 8084

# Используем entrypoint для запуска миграций и сервиса
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/search-service"]
