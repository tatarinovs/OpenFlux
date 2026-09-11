# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o universal-bypass-tool .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache iptables iproute2 ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/universal-bypass-tool /app/universal-bypass-tool
COPY deploy/docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh /app/universal-bypass-tool

ENTRYPOINT ["/app/docker-entrypoint.sh"]
