package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/client"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
)

type RaftNode struct {
	NH             *dragonboat.NodeHost
	ShardID        uint64
	ReplicaID      uint64
	SelfMember     Member
	InitialMembers []Member
	Join           bool
	Roles          []NodeRole
	stateMachine   func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine
}

func (mn *RaftNode) StartReplica() error {

	cfg := config.Config{
		ReplicaID:          mn.ReplicaID,
		ShardID:            mn.ShardID,
		CheckQuorum:        true,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		SnapshotEntries:    1000,
		CompactionOverhead: 500,
		IsNonVoting:        !ContainsRole(mn.Roles, RoleConsensus),
	}

	// stateMachine := func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
	// 	return NewMasterKVRocksDBStateMachine(clusterID, nodeID)
	// }

	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		return err
	}

	fmt.Println(base_path + "/wal/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + strconv.Itoa(mn.SelfMember.Port))
	mn.NH, err = dragonboat.NewNodeHost(config.NodeHostConfig{
		WALDir:         base_path + "/wal/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + mn.SelfMember.IP + "-" + strconv.Itoa(mn.SelfMember.Port),
		NodeHostDir:    base_path + "/node/" + strconv.FormatUint(mn.ReplicaID, 10) + "/" + mn.SelfMember.IP + "-" + strconv.Itoa(mn.SelfMember.Port),
		RTTMillisecond: 200,
		RaftAddress:    MemmberToAddr(mn.SelfMember),
	})
	if err != nil {
		return err
	}

	if !mn.Join && !IsMemberInMemberArray(mn.SelfMember, mn.InitialMembers) {
		return errors.New("the node itself must be inside initial-members")
	}

	initialMembersMap := ToInitialMembersMap(mn.InitialMembers)
	if mn.Join {
		initialMembersMap = map[uint64]string{}
	}
	return mn.NH.StartOnDiskReplica(initialMembersMap, mn.Join, mn.stateMachine, cfg)

}

func (mn *RaftNode) RequestAddReplica(replicaID uint64, member Member) error {
	addr := MemmberToAddr(member)
	rs, err := mn.NH.RequestAddReplica(mn.ShardID, replicaID, addr, 0, 3*time.Second)
	select {
	case r := <-rs.ResultC():
		if r.Completed() {
			fmt.Println("✅ Réplica añadida exitosamente")
		} else {
			fmt.Printf("❌ Error añadiendo réplica: Result=%v", r)
		}
	}
	return err
}

func (mn *RaftNode) GetClient() *client.Session {
	return mn.NH.GetNoOPSession(mn.ShardID)
}

func InitRaftNode(ShardID uint64, ReplicaID uint64, selfMember Member, initialMembers []Member, join bool, roles []NodeRole, stateMachineFn func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine) (*RaftNode, error) {
	raftNode := &RaftNode{}
	raftNode.ReplicaID = ReplicaID
	raftNode.ShardID = ShardID
	raftNode.SelfMember = selfMember
	raftNode.InitialMembers = initialMembers
	raftNode.Join = join
	raftNode.Roles = roles
	raftNode.stateMachine = func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewMasterKVRocksDBStateMachine(clusterID, nodeID)
	}
	err := raftNode.StartReplica()
	if err != nil {
		return nil, err
	}

	return raftNode, nil
}
