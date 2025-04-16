package server

import (
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
)

type ListenerFunc func(network, address string) (net.Listener, error)

type GRPCServer interface {
	Serve(lis net.Listener) error
	GracefulStop()
}

type GRPCServerFactory func() GRPCServer

func DefaultListener(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func DefaultGRPCServerFactory() GRPCServer {
	return grpc.NewServer()
}

func StartGRPC(
	config config.Config,
	listen ListenerFunc,
	gprcServerFactory GRPCServerFactory,
) error {

	port := config.Port

	if !utils.IsValidPort(port) {
		return fmt.Errorf("invalid 'port' in config: '%d'", port)
	}

	lis, err := listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := gprcServerFactory()
	defer s.GracefulStop()

	log.Printf("🚀 gRPC server listening at :%d\n", port)
	return s.Serve(lis)
}
