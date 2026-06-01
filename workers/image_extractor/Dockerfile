# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app/workers/image_extractor

# Install build dependencies
RUN apk add --no-cache git

# Copy shared telemetry package
COPY shared/go/telemetry/ /app/shared/go/telemetry/

# Copy dependencies and source
COPY workers/image_extractor/go.mod workers/image_extractor/go.sum ./
RUN go mod download

# Copy source code
COPY workers/image_extractor/main.go .
COPY workers/image_extractor/config/ config/
COPY workers/image_extractor/domain/ domain/
COPY workers/image_extractor/repositories/ repositories/
COPY workers/image_extractor/services/ services/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o image-extractor main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/workers/image_extractor/image-extractor .

# Run the binary
CMD ["./image-extractor"]
