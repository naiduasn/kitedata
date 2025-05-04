FROM golang:1.22-alpine as builder

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o kitedata ./cmd

# Use a smaller image for the final build
FROM alpine:latest

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/kitedata /app/kitedata

# Copy example config
COPY config.yaml.example /app/config.yaml.example

# Create directories for data
RUN mkdir -p /app/historical_data /app/parquet_data

# Set the entrypoint
ENTRYPOINT ["/app/kitedata"]

# Default command (can be overridden)
CMD ["--help"]