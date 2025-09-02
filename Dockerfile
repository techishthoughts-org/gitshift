# GitPersona Dockerfile
# Multi-stage build for optimized production image

# Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gitpersona .

# Production stage
FROM alpine:latest

# Install git, openssh-client for git operations
RUN apk --no-cache add git openssh-client ca-certificates tzdata bash zsh

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /home/appuser

# Copy the binary from builder stage
COPY --from=builder /app/gitpersona /usr/local/bin/gitpersona

# Make binary executable
RUN chmod +x /usr/local/bin/gitpersona

# Create config directory
RUN mkdir -p /home/appuser/.config/gitpersona && \
    mkdir -p /home/appuser/.ssh && \
    chown -R appuser:appgroup /home/appuser

# Switch to non-root user
USER appuser

# Set environment variables
ENV PATH="/usr/local/bin:${PATH}"
ENV HOME="/home/appuser"

# Create a sample configuration
RUN gitpersona --help > /dev/null 2>&1 || true

# Set default command
ENTRYPOINT ["gitpersona"]
CMD ["--help"]

# Labels for better maintainability
LABEL maintainer="arthur.alvesdeveloper@gmail.com"
LABEL version="2.0.0"
LABEL description="GitPersona - GitHub Identity Manager TUI"
LABEL org.opencontainers.image.source="https://github.com/thukabjj/GitPersona"
