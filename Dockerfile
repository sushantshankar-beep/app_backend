FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 👇 build CRON binary instead of API server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o cron ./cmd/cron/main.go


# ============================
# RUNTIME
# ============================

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

# 👇 copy cron binary only
COPY --from=builder /app/cron .

CMD ["./cron"]
