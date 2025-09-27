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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitshift .

# Final stage
FROM alpine:latest

# Install necessary packages
RUN apk --no-cache add ca-certificates git openssh-client

# Create non-root user
RUN addgroup -g 1001 -S gitshift && \
    adduser -u 1001 -S gitshift -G gitshift

# Set working directory
WORKDIR /home/gitshift

# Copy the binary from builder stage
COPY --from=builder /app/gitshift /usr/local/bin/gitshift

# Change ownership
RUN chown -R gitshift:gitshift /home/gitshift

# Switch to non-root user
USER gitshift

# Set up SSH directory
RUN mkdir -p ~/.ssh && chmod 700 ~/.ssh

# Create config directory
RUN mkdir -p ~/.config/gitshift

# Default command
ENTRYPOINT ["gitshift"]
CMD ["--help"]
