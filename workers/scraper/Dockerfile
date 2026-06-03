# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app/workers/scraper

# Copy shared telemetry package
COPY shared/go/telemetry/ /app/shared/go/telemetry/

# Copy the scraper worker dependencies and source
COPY workers/scraper/go.mod workers/scraper/go.sum ./
RUN go mod download

COPY workers/scraper/main.go .
COPY workers/scraper/config/ config/
COPY workers/scraper/domain/ domain/
COPY workers/scraper/repositories/ repositories/
COPY workers/scraper/services/ services/

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/workers/scraper/main .

CMD ["./main"]
