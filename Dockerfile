FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/app/main.go


FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y \
    wkhtmltopdf \
    chromium \
    ca-certificates \
    fontconfig \
    fonts-dejavu \
    fonts-liberation \
    fonts-freefont-ttf \
    && rm -rf /var/lib/apt/lists/*

# 👇 copy binary
COPY --from=builder /app/server .

# 👇 templates + storage
COPY --from=builder /app/internal/template ./internal/template
COPY --from=builder /app/internal/storage ./internal/storage

ENV CHROME_BIN=/usr/bin/chromium
ENV WKHTMLTOPDF_BIN=/usr/bin/wkhtmltopdf

EXPOSE 8080
CMD ["./server"]
