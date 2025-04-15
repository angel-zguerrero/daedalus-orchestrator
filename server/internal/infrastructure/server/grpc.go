package server

import (
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"log"
	"net"
	"strconv"

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
	config map[string]string,
	listen ListenerFunc,
	gprcServerFactory GRPCServerFactory,
) error {
	const defaultPort = 50052
	port := defaultPort

	if val, ok := config["port"]; ok {
		p, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		port = p
	}

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
