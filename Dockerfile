FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/app/main.go


FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates \
    chromium \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    fontconfig

# 👇 copy binary
COPY --from=builder /app/server .

# 👇 copy templates + storage folders
COPY --from=builder /app/internal/template ./internal/template
COPY --from=builder /app/internal/storage ./internal/storage

EXPOSE 8080
CMD ["./server"]
