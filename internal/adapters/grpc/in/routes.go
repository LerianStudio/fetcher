package in

import (
	command "golang-plugin-boilerplate/internal/services/command"
	query "golang-plugin-boilerplate/internal/services/query"
	proto "golang-plugin-boilerplate/pkg/proto/example"

	libLog "github.com/LerianStudio/lib-commons/commons/log"
	libHTTP "github.com/LerianStudio/lib-commons/commons/net/http"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/go-playground/validator.v9"
)

// NewRouterGRPC registers routes to the grpc.
func NewRouterGRPC(lg libLog.Logger, tl *libOtel.Telemetry, exq *query.ExampleQuery, exc *command.ExampleCommand) *grpc.Server {
	tlMid := libHTTP.NewTelemetryMiddleware(tl)

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			tlMid.WithTelemetryInterceptor(tl),
			libHTTP.WithGrpcLogging(libHTTP.WithCustomLogger(lg)),
			tlMid.EndTracingSpansInterceptor(),
		),
	)

	reflection.Register(server)

	exampleProto := &ExampleProto{
		ExampleQuery:   exq,
		ExampleCommand: exc,
		Validator:      validator.New(),
	}

	proto.RegisterExampleServer(server, exampleProto)

	return server
}
