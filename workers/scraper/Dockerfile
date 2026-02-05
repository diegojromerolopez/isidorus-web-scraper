FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod ./
# RUN go mod download 
# Simplified for this environment, ideally run go mod tidy inside if creating fresh

COPY . .

# Since we manually created go.mod, we might need to verify dependencies.
# For local development within docker, usually we run tidy.
RUN go mod tidy
RUN go build -o main .

CMD ["./main"]
