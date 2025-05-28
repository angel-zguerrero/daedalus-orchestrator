package dragonboat_test

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"errors"
	"reflect"
	"testing"

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
		createStateMachine dragonboat.CreateStateMachineFunc,
	) (*dragonboat.RaftNode, error) {
		calledWithShardID = shardID
		calledWithReplicaID = replicaID
		calledWithSelfMember = selfMember
		calledWithInitialMembers = initialMembers
		calledWithJoin = join
		calledWithRoles = roles
		calledWithStateMachineConstr = createStateMachine // Store the function itself

		// Return dummy values for the test
		return &dragonboat.RaftNode{}, nil // Or an error if you want to test error propagation
	}

	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	// Sample arguments for InitMasterNode
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

	// Compare function pointers using reflect
	// This is the most reliable way to check if the correct constructor was passed.
	expectedStateMachineConstr := reflect.ValueOf(dragonboat.NewMasterKVRocksDBStateMachine)
	actualStateMachineConstr := reflect.ValueOf(calledWithStateMachineConstr)
	assert.Equal(t, expectedStateMachineConstr.Pointer(), actualStateMachineConstr.Pointer(), "StateMachine constructor should be NewMasterKVRocksDBStateMachine")
}

// Test for when InitRaftNodeFunc returns an error
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
		createStateMachine dragonboat.CreateStateMachineFunc,
	) (*dragonboat.RaftNode, error) {
		return nil, expectedError
	}

	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitMasterNode(1, dragonboat.Member{}, nil, false, nil)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
}

// Test for when NewMasterKVRocksDBStateMachine is passed (as a type check, not pointer comparison)
// This is a weaker check than pointer comparison but can be useful for sanity.
func TestMasterNode_PassesCorrectStateMachineType(t *testing.T) {
	originalInitRaftNodeFunc := dragonboat.InitRaftNodeFunc
	defer func() {
		dragonboat.InitRaftNodeFunc = originalInitRaftNodeFunc
	}()

	var passedCreateFunc dragonboat.CreateStateMachineFunc

	mockInitRaftNode := func(
		shardID uint64,
		replicaID uint64,
		selfMember dragonboat.Member,
		initialMembers []dragonboat.Member,
		join bool,
		roles []dragonboat.NodeRole,
		createStateMachine dragonboat.CreateStateMachineFunc,
	) (*dragonboat.RaftNode, error) {
		passedCreateFunc = createStateMachine
		return &dragonboat.RaftNode{}, nil
	}
	dragonboat.InitRaftNodeFunc = mockInitRaftNode

	_, err := dragonboat.InitMasterNode(1, dragonboat.Member{}, nil, false, nil)
	require.NoError(t, err)

	// Create dummy instances to check if the passed function behaves as expected
	// This isn't a perfect type check at compile time, but it verifies the signature at runtime.
	// Note: This part of the test will fail if NewMasterKVRocksDBStateMachine panics or errors
	// due to uninitialized global config or other setup issues not handled here.
	// The reflection check in TestMasterNode_CallsInitRaftNodeCorrectly is more robust for function identity.
	if passedCreateFunc != nil {
		sm := passedCreateFunc(1,1) // Call it with dummy values
		_, ok := sm.(statemachine.IOnDiskStateMachine) // Check if it returns the expected interface
		assert.True(t, ok, "Passed function should return an IOnDiskStateMachine")

		// Further check if it's specifically a *KVBaseRocksDBStateMachine which NewMasterKVRocksDBStateMachine returns wrapped
		// This is tricky because NewMasterKVRocksDBStateMachine returns IOnDiskStateMachine,
		// and the actual type is *KVBaseRocksDBStateMachine after type assertion in tests.
		// The NewKVStateMachine wraps it. The most direct check is the reflect.ValueOf().Pointer() comparison.
	} else {
		t.Error("InitRaftNode was not called with a CreateStateMachineFunc")
	}
}
