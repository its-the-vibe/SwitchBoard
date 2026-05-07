# Build stage
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o switchboard .

# Runtime stage
FROM scratch

# Copy binary from builder
COPY --from=builder /app/switchboard .

# Copy static files and config
COPY static ./static

# Expose port
EXPOSE 8080

# Run the application
CMD ["./switchboard"]
