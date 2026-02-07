# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum
COPY workers/image_extractor/go.mod workers/image_extractor/go.sum ./
RUN go mod download

# Copy source code
COPY workers/image_extractor/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o image-extractor main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/image-extractor .

# Run the binary
CMD ["./image-extractor"]
