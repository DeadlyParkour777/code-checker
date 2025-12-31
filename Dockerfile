FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

ARG SERVICE_DIR

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go work init && go work use -r . && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /main ./$SERVICE_DIR/cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /main .

CMD ["./main"]
