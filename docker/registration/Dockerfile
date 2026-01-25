# Stage 1: Build the Go binary
FROM golang:1.25.5-alpine AS builder

WORKDIR /build

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy source code

COPY . .

# Build the binary with static linking and strip debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-s -w -extldflags "-static"' \
    -o registration-server \
    ./cmd/registration-server


# Stage 2: Create minimal image runtime
FROM alpine:latest

# Add CA certifications for HTTPS calls
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/registration-server .

# Use non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose port
EXPOSE 8080

# Run the server
CMD ["./registration-server"]