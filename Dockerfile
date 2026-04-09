# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o chatapi ./cmd/chatapi

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/chatapi .

# Change ownership
RUN chown appuser:appgroup chatapi

# Create persistent data directory for SQLite and set ownership
RUN mkdir -p /data && chown appuser:appgroup /data

# Switch to non-root user
USER appuser

# Declare volume for SQLite data
VOLUME ["/data"]

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./chatapi"]