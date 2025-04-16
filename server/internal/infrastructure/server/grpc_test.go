package server_test

import (
	"deadalus-orch/server/internal/infrastructure/server"
	"deadalus-orch/server/internal/pkg/config"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock del GRPC server
type MockGRPCServer struct {
	mock.Mock
}

func (m *MockGRPCServer) Serve(lis net.Listener) error {
	args := m.Called(lis)
	return args.Error(0)
}

func (m *MockGRPCServer) GracefulStop() {
	m.Called()
}

func makeListener(t *testing.T, addr string) net.Listener {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	return lis
}

func TestStartGRPC(t *testing.T) {
	tests := []struct {
		name          string
		config        config.Config
		listenFunc    server.ListenerFunc
		newServerFunc server.GRPCServerFactory
		expectError   bool
	}{
		{
			name: "custom valid port",
			config: config.Config{
				Port: 5050,
			},
			listenFunc: func(network, address string) (net.Listener, error) {
				return makeListener(t, ":5050"), nil
			},
			newServerFunc: func() server.GRPCServer {
				mockSrv := new(MockGRPCServer)
				mockSrv.On("Serve", mock.Anything).Return(nil)
				mockSrv.On("GracefulStop").Return()
				return mockSrv
			},
			expectError: false,
		},
		{
			name: "port out of range",
			config: config.Config{
				Port: 99999,
			},
			listenFunc: func(network, address string) (net.Listener, error) {
				t.Fatal("should not call listen when port is invalid")
				return nil, nil
			},
			newServerFunc: nil,
			expectError:   true,
		},
		{
			name:   "listen fails",
			config: config.Config{},
			listenFunc: func(network, address string) (net.Listener, error) {
				return nil, errors.New("mock listen failure")
			},
			newServerFunc: nil,
			expectError:   true,
		},
		{
			name:   "serve fails",
			config: config.Config{},
			listenFunc: func(network, address string) (net.Listener, error) {
				return makeListener(t, ":50052"), nil
			},
			newServerFunc: func() server.GRPCServer {
				mockSrv := new(MockGRPCServer)
				mockSrv.On("Serve", mock.Anything).Return(errors.New("serve failed"))
				mockSrv.On("GracefulStop").Return()
				return mockSrv
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.StartGRPC(tt.config, tt.listenFunc, tt.newServerFunc)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
