# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app/workers/writer

# Copy shared telemetry package
COPY shared/go/telemetry/ /app/shared/go/telemetry/

# Copy writer dependencies and source
COPY workers/writer/go.mod workers/writer/go.sum ./
RUN go mod download

COPY workers/writer/main.go .
COPY workers/writer/config/ config/
COPY workers/writer/domain/ domain/
COPY workers/writer/models/ models/
COPY workers/writer/repositories/ repositories/
COPY workers/writer/services/ services/

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/workers/writer/main .

CMD ["./main"]
