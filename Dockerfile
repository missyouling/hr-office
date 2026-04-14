# Multi-stage build for 人事行政管理系统 (hr-office)
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY backend/go.mod backend/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY backend/ ./

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o siapp .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata sqlite && \
    mkdir /app && \
    addgroup -g 1000 siapp && \
    adduser -u 1000 -G siapp -s /bin/sh -D siapp

# Set timezone
ENV TZ=Asia/Shanghai

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/siapp .

# Create data directory for SQLite (if used)
RUN mkdir -p /app/data && chown -R siapp:siapp /app

# Switch to non-root user
USER siapp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default environment variables
ENV SIAPP_ADDR=0.0.0.0:8080
ENV SIAPP_DATABASE_TYPE=sqlite
ENV SIAPP_DATABASE_PATH=/app/data/siapp.db

# Run the application
CMD ["./siapp"]
