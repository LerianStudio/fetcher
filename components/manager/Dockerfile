# syntax=docker/dockerfile:1.4 

FROM golang:1.26-alpine AS builder

# Install required packages: git is needed for private modules
RUN apk add --no-cache git

# Set the working directory for the build
WORKDIR /fetcher

# Copy only go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./

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
FROM alpine:3.23

WORKDIR /app

# Install certificates/timezone data and curl for health checks
RUN apk add --no-cache zlib ca-certificates tzdata curl && \
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
