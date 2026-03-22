FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app/ranking-service

COPY ranking-service/go.mod ranking-service/go.sum ./
COPY pkg/pb ../pkg/pb
COPY pkg/utils ../pkg/utils

RUN go mod download

COPY ranking-service/. .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/ranking-service ./cmd/service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata dos2unix

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /bin/ranking-service /app/ranking-service
COPY --from=builder /bin/migrate /app/migrate

COPY ranking-service/entrypoint.sh /app/entrypoint.sh
RUN dos2unix /app/entrypoint.sh && chmod +x /app/entrypoint.sh

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8087

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/ranking-service"]

