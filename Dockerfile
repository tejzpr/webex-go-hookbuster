# ── Build stage ───────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN go build -o /hookbuster .

# ── Runtime stage ────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /hookbuster /usr/local/bin/hookbuster

# Environment variables for deployment mode:
#   TOKEN  - Webex access token (required)
#   PORT   - Target forwarding port (required)
#   TARGET - Target hostname/IP (default: localhost)

ENTRYPOINT ["hookbuster"]
