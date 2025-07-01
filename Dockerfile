# Build stage
FROM golang:1.23-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimization flags
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/issue-tracker ./cmd/

# Final lightweight runtime stage
FROM alpine:3.18

# Add CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /app

# Create a non-root user to run the application
RUN adduser -D -g '' appuser
USER appuser

# Copy binary from builder stage
COPY --from=builder /app/issue-tracker .

# Expose ports
# HTTP port
EXPOSE 8080
# GRPC port
EXPOSE 50052

# Command to run the executable
CMD ["./issue-tracker"]