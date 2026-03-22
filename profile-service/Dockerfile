# syntax=docker/dockerfile:1.7
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app/profile-service

# Copy profile-service go.mod/go.sum and local modules required by replace directives.
COPY profile-service/go.mod profile-service/go.sum ./
COPY pkg/pb ../pkg/pb
COPY pkg/utils ../pkg/utils

# Download dependencies.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

# Copy profile-service source code.
COPY profile-service/. .

# Build service and migration binaries.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/profile-service ./cmd/service/main.go
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/migrate ./cmd/migrate/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata dos2unix

RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /bin/profile-service /app/profile-service
COPY --from=builder /bin/migrate /app/migrate

COPY profile-service/entrypoint.sh /app/entrypoint.sh
RUN dos2unix /app/entrypoint.sh && chmod +x /app/entrypoint.sh

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8085

ENTRYPOINT ["/app/entrypoint.sh"]
CMD ["/app/profile-service"]

