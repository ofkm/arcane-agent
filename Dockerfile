# Multi-stage build for smaller image
FROM golang:1.24.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version information
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

# Build the binary with version information
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static' \
    -X github.com/ofkm/arcane-agent/internal/version.Version=${VERSION} \
    -X github.com/ofkm/arcane-agent/internal/version.Commit=${COMMIT} \
    -X github.com/ofkm/arcane-agent/internal/version.Date=${DATE}" \
    -o arcane-agent ./cmd/agent

# Final stage
FROM scratch

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/arcane-agent /arcane-agent

# Expose default port (adjust as needed)
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/arcane-agent"]