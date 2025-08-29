FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

ARG SERVICE_DIR

RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./$SERVICE_DIR/cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /main .

CMD ["./main"]