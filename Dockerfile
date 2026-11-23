FROM golang:1.27.4-alpine3.24 AS builder

WORKDIR /src
ENV GOTOOLCHAIN=local

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY cmd/ cmd/
COPY internal/ internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -mod=readonly -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.24.4

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S -g 1001 appgroup && adduser -S -D -H -u 1001 -G appgroup appuser

WORKDIR /app
COPY --from=builder /out/api /app/api
USER appuser

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --quiet --timeout=2 -O /dev/null http://127.0.0.1:8080/ready
ENTRYPOINT ["/app/api"]
