# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:3.19

# Set working directory to /app
WORKDIR /app

# Install CA certificates
RUN apk --no-cache add ca-certificates tzdata

# Create necessary directories
RUN mkdir -p /app/docs

# Copy the binary from builder
COPY --from=builder /app/server .

# Copy configuration file to expected location
COPY docs/payer_status.yaml /app/docs/

# Copy test client files
COPY test/ /app/test/

# Expose ports
EXPOSE 8080 9090

# Set environment variables with defaults
ENV PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info \
    CONFIG_PATH=./docs/payer_status.yaml

# Run the server
CMD ["/app/server", "--config", "/app/config/payer_status.yaml"]
