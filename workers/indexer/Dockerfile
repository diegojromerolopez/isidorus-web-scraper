# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app/workers/indexer

# Copy shared telemetry package
COPY shared/go/telemetry/ /app/shared/go/telemetry/

# Copy dependencies and source
COPY workers/indexer/go.mod workers/indexer/go.sum ./
RUN go mod download

COPY workers/indexer/main.go .
COPY workers/indexer/config/ config/
COPY workers/indexer/domain/ domain/
COPY workers/indexer/repositories/ repositories/
COPY workers/indexer/services/ services/

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/workers/indexer/main .

CMD ["./main"]
