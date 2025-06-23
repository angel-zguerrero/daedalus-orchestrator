package dragonboat_test

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/lni/dragonboat/v4/statemachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMasterNode_CallsInitRaftNodeCorrectly(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitRaftNodeFunc
	defer func() {
		dragonboat.InitRaftNodeFunc = originalInitRaftNodeFunc
	}()

	var (
		calledWithShardID            uint64
		calledWithReplicaID          uint64
		calledWithSelfMember         dragonboat.Member
		calledWithInitialMembers     []dragonboat.Member
		calledWithJoin               bool
		calledWithRoles              []dragonboat.NodeRole
		calledWithStateMachineConstr interface{} // Store as interface{} to compare function pointers later
	)

	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		calledWithShardID = shardID
		calledWithReplicaID = replicaID
		calledWithSelfMember = selfMember
		calledWithInitialMembers = initialMembers
		calledWithJoin = join
		calledWithRoles = roles
		calledWithStateMachineConstr = createStateMachine

		return &dragonboat.RaftNode{}, nil
	}

	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	testReplicaID := uint64(1)
	testSelfMember := dragonboat.Member{IP: "127.0.0.1", Port: 1234}
	testInitialMembers := []dragonboat.Member{{IP: "127.0.0.1", Port: 1234}, {IP: "127.0.0.2", Port: 1235}}
	testJoin := false
	testRoles := []dragonboat.NodeRole{dragonboat.RoleConsensus, dragonboat.RoleScheduler}

	_, err := dragonboat.InitMasterNode(testReplicaID, testSelfMember, testInitialMembers, testJoin, testRoles)
	require.NoError(t, err)

	assert.Equal(t, uint64(dragonboat.MasterShardID), calledWithShardID, "ShardID should be MasterShardID")
	assert.Equal(t, testReplicaID, calledWithReplicaID, "ReplicaID should match")
	assert.Equal(t, testSelfMember, calledWithSelfMember, "SelfMember should match")
	assert.Equal(t, testInitialMembers, calledWithInitialMembers, "InitialMembers should match")
	assert.Equal(t, testJoin, calledWithJoin, "Join flag should match")
	assert.Equal(t, testRoles, calledWithRoles, "Roles should match")

	expectedStateMachineConstr := reflect.ValueOf(dragonboat.NewMasterKVStateMachine)
	actualStateMachineConstr := reflect.ValueOf(calledWithStateMachineConstr)
	assert.Equal(t, expectedStateMachineConstr.Pointer(), actualStateMachineConstr.Pointer(), "StateMachine constructor should be NewMasterKVRocksDBStateMachine")
}

func TestMasterNode_InitRaftNodeErrorPropagation(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitRaftNodeFunc
	defer func() {
		dragonboat.InitRaftNodeFunc = originalInitRaftNodeFunc
	}()

	expectedError := errors.New("raft node init failed")
	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		return nil, expectedError
	}

	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitMasterNode(1, dragonboat.Member{}, nil, false, nil)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

func TestMasterNode_PassesCorrectStateMachineType(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitRaftNodeFunc
	defer func() {
		dragonboat.InitRaftNodeFunc = originalInitRaftNodeFunc
	}()

	var passedCreateFunc func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine

	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		createStateMachine func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine,
	) (*dragonboat.RaftNode, error) {
		fmt.Println("esto se debe llamar?????", createStateMachine != nil)
		passedCreateFunc = createStateMachine
		return &dragonboat.RaftNode{}, nil
	}
	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitMasterNode(1, dragonboat.Member{}, nil, false, nil)
	require.NoError(t, err)
	time.Sleep(2 * time.Second)
	if passedCreateFunc != nil {
		fmt.Println("passedCreateFunc !!!!")
		//fmt.Println(passedCreateFunc)
		sm := passedCreateFunc(1, 1)
		_, ok := sm.(statemachine.IOnDiskStateMachine)
		assert.True(t, ok, "Passed function should return an IOnDiskStateMachine")
	} else {
		t.Error("InitRaftNode was not called with a CreateStateMachineFunc")
	}
}
