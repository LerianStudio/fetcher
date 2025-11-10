FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /golang-plugin-boilerplate

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETPLATFORM
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo $TARGETPLATFORM | cut -d'/' -f2) go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o /app cmd/app/main.go

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app /app

COPY --from=builder /golang-plugin-boilerplate/migrations /migrations

EXPOSE 3005 7001

ENTRYPOINT ["/app"]