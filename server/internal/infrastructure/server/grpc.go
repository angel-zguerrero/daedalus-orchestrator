package server

import (
	"deadalus-orch/server/internal/pkg/utils"
	"fmt"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"
)

func StartGRPC(config map[string]string, db interface{}) error {
	const defaultPort = 50052
	port := defaultPort

	if val, ok := config["port"]; ok {
		if p, err := strconv.Atoi(val); err == nil {
			port = p
		}
	}

	if !utils.IsValidPort(port) {
		fmt.Printf("❌ Invalid 'port' in config: '%d'. Using fallback.\n", port)
		port = defaultPort
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	defer s.GracefulStop()

	log.Printf("🚀 gRPC server listening at :%d\n", port)
	return s.Serve(lis)
}
