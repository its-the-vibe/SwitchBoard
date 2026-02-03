# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o switchboard .

# Runtime stage
FROM alpine:latest

RUN apk update && apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/switchboard .

# Copy static files and config
COPY static ./static
COPY config.json .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./switchboard"]
