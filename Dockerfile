# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitpersona .

# Final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates git openssh-client

# Create non-root user
RUN addgroup -g 1001 -S gitpersona && \
    adduser -u 1001 -S gitpersona -G gitpersona

# Set working directory
WORKDIR /home/gitpersona

# Copy the binary from builder stage
COPY --from=builder /app/gitpersona /usr/local/bin/gitpersona

# Change ownership
RUN chown -R gitpersona:gitpersona /home/gitpersona

# Switch to non-root user
USER gitpersona

# Set up SSH directory
RUN mkdir -p ~/.ssh && chmod 700 ~/.ssh

# Create config directory
RUN mkdir -p ~/.config/gitpersona

# Default command
ENTRYPOINT ["gitpersona"]
CMD ["--help"]