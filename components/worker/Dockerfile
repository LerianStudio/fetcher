# syntax=docker/dockerfile:1.4

FROM golang:1.26-alpine AS builder

# Install required packages: git is needed for private modules (e.g., from GitHub)
RUN apk add --no-cache git


# Set the working directory for the build
WORKDIR /fetcher

# Copy only go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./

# Download Go modules using GitHub token from BuildKit secret
RUN --mount=type=secret,id=github_token \
  GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
  printf "machine github.com\nlogin x-oauth-basic\npassword %s\n" "$GITHUB_TOKEN" > ~/.netrc && \
  go mod download && rm ~/.netrc

# Copy only necessary source files for worker component
COPY pkg/ pkg/
COPY components/worker/ components/worker/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo \
  -ldflags='-w -s -extldflags "-static"' \
  -o /worker ./components/worker/cmd/app/main.go

# Stage 2: Final minimal image with only the binary
FROM alpine:latest

WORKDIR /app

# Create a non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S -h /app appuser -G appgroup && \
    chown -R appuser:appgroup /app

ENV HOME=/app

# Copy the binary from the builder stage
COPY --from=builder /worker /app/worker

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 4005

# Set the entrypoint to the worker binary
ENTRYPOINT ["/app/worker"]
