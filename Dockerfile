# Build stage
FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod ./
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -installsuffix cgo -o switchboard .

# Runtime stage
FROM gcr.io/distroless/static-debian13:nonroot

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/switchboard .

# Copy static files and config
COPY static ./static

# Expose port
EXPOSE 8080

USER nonroot:nonroot

# Run the application
CMD ["./switchboard"]
