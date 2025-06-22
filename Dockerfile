# Build stage
FROM golang:1.21-alpine AS builder

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

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o crawler cmd/crawler/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 crawler && \
    adduser -D -s /bin/sh -u 1001 -G crawler crawler

WORKDIR /home/crawler

# Copy binary and set ownership
COPY --from=builder /app/crawler .
COPY --from=builder /app/env.example .env
RUN chown -R crawler:crawler /home/crawler

# Switch to non-root user
USER crawler

# Expose port (if needed for future API)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD pgrep crawler || exit 1

# Run the crawler
CMD ["./crawler"]