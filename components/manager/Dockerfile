# syntax=docker/dockerfile:1.4

FROM golang:1.25-alpine AS builder

# Install required packages: git is needed for private modules
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

# Copy only necessary source files for manager component
COPY pkg/ pkg/
COPY components/manager/ components/manager/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo \
  -ldflags='-w -s -extldflags "-static"' \
  -o /manager ./components/manager/cmd/app/main.go

# Stage 2: Final image using minimal runtime dependencies
FROM alpine:3.19

WORKDIR /app

# Install certificates/timezone data required by the manager binary
RUN apk add --no-cache ca-certificates tzdata && \
    update-ca-certificates

# Create a non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Copy the binary from the builder stage
COPY --from=builder /manager /app/manager

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose the application port
EXPOSE 4006

# Set the entrypoint to the manager binary
ENTRYPOINT ["/app/manager"]

