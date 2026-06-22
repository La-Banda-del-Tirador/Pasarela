# ── Stage 1: compilar ──────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod .
COPY *.go .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o pasarela .

# ── Stage 2: imagen final mínima ───────────────────────────────────────────────
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/pasarela .

EXPOSE 9191
CMD ["./pasarela"]
