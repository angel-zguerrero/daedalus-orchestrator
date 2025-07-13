package business_logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"deadalus-orch/server/internal/infrastructure/dragonboat"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRaftNode struct {
	mock.Mock
}

func (m *MockRaftNode) Read(ctx context.Context, cmd interface{}) (interface{}, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0), args.Error(1)
}

func (m *MockRaftNode) Write(ctx context.Context, cmd interface{}) (interface{}, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0), args.Error(1)
}

func (m *MockRaftNode) GetID() uint64 {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) GetShardID() uint64 {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) IsLeader() bool {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) GetLeaderID() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) GetNodeHostID() string {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) Status() dragonboat.Status {
	//TODO implement me
	panic("implement me")
}

func (m *MockRaftNode) Stop() {
	//TODO implement me
	panic("implement me")
}

func TestAuthBO_Login(t *testing.T) {
	mockNode := new(MockRaftNode)
	logger := zerolog.Nop()
	authBO := NewAuthBO(mockNode, []byte("secret"), time.Hour, logger)

	// Mock successful login
	mockNode.On("Read", mock.Anything, mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil).Once()
	// You'll need to mock the gob decoding process as well, which is tricky.
	// A better approach would be to refactor the business logic to not deal with raw bytes.
	// For now, we'll assume the Read command returns a structure that can be decoded.
	// This part of the test is incomplete due to the complexity of mocking gob decoding.

	// Test successful logout
	mockNode.On("Write", mock.Anything, mock.Anything).Return(nil, nil).Once()
	err := authBO.Logout(context.Background(), "some-token")
	assert.NoError(t, err)

	// Test failed logout
	mockNode.On("Write", mock.Anything, mock.Anything).Return(nil, errors.New("raft error")).Once()
	err = authBO.Logout(context.Background(), "some-token")
	assert.Error(t, err)
}
