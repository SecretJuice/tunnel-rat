FROM golang:alpine AS builder

# Install WireGuard tools
RUN apk add --no-cache wireguard-tools curl jq

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum to download dependencies first
COPY go.mod ./
RUN go mod download

# Copy the entire project so internal packages can be built
COPY . .

# Build the client binary
RUN go build -o /app/client ./cmd/client/main.go

# Final image
FROM alpine

# Install WireGuard
RUN apk add --no-cache wireguard-tools

# Set working directory
WORKDIR /app

# Copy the compiled Go binary
COPY --from=builder /app/client /app/entrypoint

# Set executable permissions
RUN chmod +x /app/entrypoint

# Entrypoint
ENTRYPOINT ["/app/entrypoint"]