# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app/api-gateway

# Copy module files and local modules required by replace directives
COPY api-gateway/go.mod api-gateway/go.sum ./
COPY pkg/pb ../pkg/pb
COPY pkg/utils ../pkg/utils

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY api-gateway/. .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/api-gateway ./cmd/gateway/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /bin/api-gateway /app/api-gateway

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

CMD ["/app/api-gateway"]

