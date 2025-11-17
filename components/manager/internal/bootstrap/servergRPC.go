package bootstrap

import (
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	libCommons "github.com/LerianStudio/lib-commons/commons"
	libLog "github.com/LerianStudio/lib-commons/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/commons/opentelemetry"
)

// ServerGRPC represents the gRPC server for Ledger service.
type ServerGRPC struct {
	server       *grpc.Server
	protoAddress string
	libLog.Logger
	libOtel.Telemetry
}

// ProtoAddress returns is a convenience method to return the proto server address.
func (sgrpc *ServerGRPC) ProtoAddress() string {
	return sgrpc.protoAddress
}

// NewServerGRPC creates an instance of gRPC Server.
func NewServerGRPC(cfg *Config, server *grpc.Server, logger libLog.Logger, telemetry *libOtel.Telemetry) *ServerGRPC {
	return &ServerGRPC{
		server:       server,
		protoAddress: cfg.ProtoAddress,
		Logger:       logger,
		Telemetry:    *telemetry,
	}
}

// Run gRPC server.
func (sgrpc *ServerGRPC) Run(l *libCommons.Launcher) error {
	sgrpc.InitializeTelemetry(sgrpc.Logger)
	defer sgrpc.ShutdownTelemetry()

	defer func() {
		if err := sgrpc.Sync(); err != nil {
			sgrpc.Fatalf("Failed to sync logger: %s", err)
		}
	}()

	listen, err := net.Listen("tcp4", sgrpc.protoAddress)
	if err != nil {
		return errors.Wrap(err, "failed to listen tcp4 server")
	}

	err = sgrpc.server.Serve(listen)
	if err != nil {
		return errors.Wrap(err, "failed to run the gRPC server")
	}

	return nil
}
