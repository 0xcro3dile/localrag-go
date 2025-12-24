# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o localrag ./cmd/localrag

# Runtime stage - minimal image
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/localrag .

# Create documents directory
RUN mkdir -p /app/documents

# Expose port
EXPOSE 8080

# Run
ENTRYPOINT ["./localrag"]
CMD ["--port", "8080", "--docs", "/app/documents"]
