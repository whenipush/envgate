package grpc

import (
	envgatev1 "github.com/whenipush/envgate/gen/go/envgate/v1"
	"github.com/whenipush/envgate/internal/delivery/grpc/handler"
	"google.golang.org/grpc"
)

func RegisterServices(server *grpc.Server, envGateHandler *handler.EnvGateHandler) {
	envgatev1.RegisterEnvGateServiceServer(server, envGateHandler)
}
