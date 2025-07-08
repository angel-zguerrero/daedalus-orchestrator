package dragonboat_test

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"errors"
	"testing"
	"time"

	db4 "github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const TenantShardID = uint64(100)

func TestTenantNode_CallsInitRaftNodeCorrectly(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitTenantRaftNodeFunc
	defer func() {
		dragonboat.InitTenantRaftNodeFunc = originalInitRaftNodeFunc
	}()

	var (
		calledWithShardID        uint64
		calledWithReplicaID      uint64
		calledWithSelfMember     dragonboat.Member
		calledWithInitialMembers []dragonboat.Member
		calledWithJoin           bool
		calledWithRoles          []dragonboat.NodeRole
	)

	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		NH *db4.NodeHost,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		calledWithShardID = shardID
		calledWithReplicaID = replicaID
		calledWithSelfMember = selfMember
		calledWithInitialMembers = initialMembers
		calledWithJoin = join
		calledWithRoles = roles

		return &dragonboat.RaftNode{}, nil
	}

	dragonboat.InitTenantRaftNodeFunc = mockInitRaftNode

	testReplicaID := uint64(1)
	testSelfMember := dragonboat.Member{IP: "127.0.0.1", Port: 1234}
	testInitialMembers := []dragonboat.Member{{IP: "127.0.0.1", Port: 1234}, {IP: "127.0.0.2", Port: 1235}}
	testJoin := false
	testRoles := []dragonboat.NodeRole{dragonboat.RoleConsensus, dragonboat.RoleScheduler}

	_, err := dragonboat.InitTenantNode(TenantShardID, testReplicaID, testSelfMember, testInitialMembers, testJoin, testRoles, nil, &dragonboat.TestPathProvider{Path: t.TempDir()})
	require.NoError(t, err)

	assert.Equal(t, TenantShardID, calledWithShardID, "ShardID should be TenantShardID")
	assert.Equal(t, testReplicaID, calledWithReplicaID, "ReplicaID should match")
	assert.Equal(t, testSelfMember, calledWithSelfMember, "SelfMember should match")
	assert.Equal(t, testInitialMembers, calledWithInitialMembers, "InitialMembers should match")
	assert.Equal(t, testJoin, calledWithJoin, "Join flag should match")
	assert.Equal(t, testRoles, calledWithRoles, "Roles should match")

}

func TestTenantNode_InitRaftNodeErrorPropagation(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitTenantRaftNodeFunc
	defer func() {
		dragonboat.InitTenantRaftNodeFunc = originalInitRaftNodeFunc
	}()

	expectedError := errors.New("raft node init failed")
	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		NH *db4.NodeHost,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		return nil, expectedError
	}

	dragonboat.InitTenantRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitTenantNode(TenantShardID, 1, dragonboat.Member{}, nil, false, nil, nil, &dragonboat.TestPathProvider{Path: t.TempDir()})
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestTenantNode_PassesCorrectStateMachineType(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitTenantRaftNodeFunc
	defer func() {
		dragonboat.InitTenantRaftNodeFunc = originalInitRaftNodeFunc
	}()

	var passedCreateFunc func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine

	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		NH *db4.NodeHost,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		passedCreateFunc = createStateMachine
		return &dragonboat.RaftNode{}, nil
	}
	dragonboat.InitTenantRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitTenantNode(TenantShardID, 1, dragonboat.Member{}, nil, false, nil, nil, &dragonboat.TestPathProvider{Path: t.TempDir()})
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	if passedCreateFunc != nil {
		sm := passedCreateFunc(1, 1)
		_, ok := sm.(statemachine.IOnDiskStateMachine)
		assert.True(t, ok, "Passed function should return an IOnDiskStateMachine")
	} else {
		t.Error("InitRaftNode was not called with a CreateStateMachineFunc")
	}
}
