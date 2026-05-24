# Этап 1: Сборка
FROM golang:1.26-alpine AS builder

LABEL authors="whenipush"

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/envgated ./cmd/envgated/main.go

FROM alpine:3.20

LABEL authors="whenipush"

RUN adduser -D -H appuser

WORKDIR /app

RUN mkdir -p /app/data && chown -R appuser:appuser /app

COPY --from=builder /app/envgated .

USER appuser

ENTRYPOINT ["./envgated"]