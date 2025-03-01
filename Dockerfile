FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o mdsrenderer ./go/server.go

# Use a smaller base image for the final container
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates libc6-compat

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/mdsrenderer /app/

# Set environment variables for terminal support
ENV TERM=xterm-256color

# Set default TTY device
# Use /dev/tty for current terminal or fallback to /dev/tty1
ENV TTY_DEVICE=/dev/tty

# Run the application with the TTY parameter
ENTRYPOINT ["/app/mdsrenderer", "--tty", "${TTY_DEVICE}"]