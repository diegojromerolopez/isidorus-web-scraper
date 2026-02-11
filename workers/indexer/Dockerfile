# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY config/ config/
COPY domain/ domain/
COPY repositories/ repositories/
COPY services/ services/

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

CMD ["./main"]
