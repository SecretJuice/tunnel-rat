FROM golang:alpine AS builder

# Install WireGuard tools
RUN apk add --no-cache wireguard-tools curl jq

# Set working directory
WORKDIR /app

# Copy and build the Go script
COPY main.go .
RUN go build -o /app/entrypoint main.go

# Final image
FROM alpine

# Install WireGuard
RUN apk add --no-cache wireguard-tools

# Set working directory
WORKDIR /app

# Copy the compiled Go binary
COPY --from=builder /app/entrypoint /app/entrypoint

# Set executable permissions
RUN chmod +x /app/entrypoint

# Entrypoint
ENTRYPOINT ["/app/entrypoint"]