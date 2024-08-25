# Build stage
FROM golang:1.23-alpine AS builder

# Install swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source from the current directory to the working Directory inside the container
COPY . .

# Generate swagger docs
RUN swag init

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:latest

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/main .

# Copy the docs directory
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./main"]