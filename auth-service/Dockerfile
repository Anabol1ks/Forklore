FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app/auth-service

# Копируем go.mod/go.sum auth-service и локальные модули, нужные для replace
COPY auth-service/go.mod auth-service/go.sum ./
COPY pkg/pb ../pkg/pb
COPY pkg/utils ../pkg/utils

# Загружаем зависимости
RUN go mod download

# Копируем исходный код auth-service
COPY auth-service/. .

# Собираем бинарники для всех компонентов
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/auth-service ./cmd/service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate/main.go

FROM alpine:latest

# Устанавливаем сертификаты для HTTPS, timezone data и dos2unix
RUN apk --no-cache add ca-certificates tzdata dos2unix

# Создаем непривилегированного пользователя
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Копируем бинарники из builder
COPY --from=builder /bin/auth-service /app/auth-service
COPY --from=builder /bin/migrate /app/migrate

# Копируем entrypoint скрипт и конвертируем окончания строк
COPY auth-service/entrypoint.sh /app/entrypoint.sh
RUN dos2unix /app/entrypoint.sh && chmod +x /app/entrypoint.sh

# Меняем владельца файлов
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Экспонируем порт gRPC
EXPOSE 8081

# Используем entrypoint для запуска миграций и сервиса
ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/auth-service"]
