# =========================
# 1) Build stage
# =========================
FROM golang:1.23-alpine AS builder

WORKDIR /src

# Ensure CA certs are present (good practice)
RUN apk add --no-cache ca-certificates

# Pre-download modules (better cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a statically-linked Linux binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/kabsa-api ./cmd/api

# =========================
# 2) Runtime stage
# =========================
FROM gcr.io/distroless/base-nonroot:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /out/kabsa-api /app/kabsa-api

# Default environment (can be overridden by EKS env vars)
ENV KABSA_ENV=Production

# Match your cfg.HTTP.Port default; expose for clarity only
EXPOSE 8080

# Distroless nonroot user already exists
USER nonroot:nonroot

ENTRYPOINT ["/app/kabsa-api"]