FROM --platform=$BUILDPLATFORM golang:1.23.2-alpine AS builder

WORKDIR /fetcher-app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETPLATFORM
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) go build -tags netgo -ldflags '-s -w -extldflags "-static"' -o /app cmd/app/main.go

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app /app
COPY --from=builder /fetcher-app/migrations /migrations

EXPOSE 4000 7001

ENTRYPOINT ["/app"]