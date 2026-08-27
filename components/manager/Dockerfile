# syntax=docker/dockerfile:1.4 

FROM golang:1.27.0-alpine AS builder

# Install required packages: git is needed for private modules
RUN apk add --no-cache git

# Set the working directory for the build
WORKDIR /fetcher

# Copy only go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
# The engine is a separate module. Before the first pkg/engine/v* tag exists, the root
# go.mod pins it via a transient `replace => ./pkg/engine`, so its go.mod must be present
# for `go mod download` to resolve the replace (the full source is copied with pkg/ below).
# Post-release the require resolves from the proxy and this copy is a harmless no-op.
COPY pkg/engine/go.mod ./pkg/engine/go.mod

# Download Go modules, using the GitHub token secret only when it is available.
RUN --mount=type=secret,id=github_token,required=false \
  if [ -f /run/secrets/github_token ] && [ -s /run/secrets/github_token ]; then \
    GITHUB_TOKEN=$(cat /run/secrets/github_token) && \
    printf "machine github.com\nlogin x-oauth-basic\npassword %s\n" "$GITHUB_TOKEN" > ~/.netrc; \
  fi && \
  go mod download && \
  rm -f ~/.netrc

# Copy only necessary source files for manager component
COPY pkg/ pkg/
COPY components/manager/ components/manager/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo \
  -ldflags='-w -s -extldflags "-static"' \
  -o /manager ./components/manager/cmd/app/main.go

# Stage 2: Final image using minimal runtime dependencies
FROM alpine:3.24.1

WORKDIR /app

# Install certificates/timezone data, patched OpenSSL runtime, and curl for health checks.
# Refresh these pins together from Alpine v3.24/main when security patches land.
RUN apk add --no-cache \
      libcrypto3=3.5.8-r0 \
      libssl3=3.5.8-r0 \
      zlib=1.3.2-r0 \
      ca-certificates=20260611-r0 \
      tzdata=2026c-r0 \
      curl=8.21.0-r0 && \
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

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:4006/health || exit 1

# Set the entrypoint to the manager binary
ENTRYPOINT ["/app/manager"]
