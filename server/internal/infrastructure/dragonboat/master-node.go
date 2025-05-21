package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"errors"
	"fmt"
	"strconv"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
)

func InitMasterNode(ReplicaID uint64, selfMember Member, initialMembers []Member, join bool) error {

	cfg := config.Config{
		ReplicaID:          ReplicaID,
		ShardID:            MasterShardID,
		CheckQuorum:        true,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		SnapshotEntries:    1000,
		CompactionOverhead: 500,
	}

	stateMachine := func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewMasterKVRocksDBStateMachine(clusterID, nodeID)
	}

	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		return err
	}

	fmt.Println(base_path + "/wal/" + strconv.FormatUint(ReplicaID, 10) + "/" + strconv.Itoa(selfMember.Port))
	nh, err := dragonboat.NewNodeHost(config.NodeHostConfig{
		WALDir:         base_path + "/wal/" + strconv.FormatUint(ReplicaID, 10) + "/" + selfMember.IP + "-" + strconv.Itoa(selfMember.Port),
		NodeHostDir:    base_path + "/node/" + strconv.FormatUint(ReplicaID, 10) + "/" + selfMember.IP + "-" + strconv.Itoa(selfMember.Port),
		RTTMillisecond: 200,
		RaftAddress:    MemmberToAddr(selfMember),
	})
	if err != nil {
		return err
	}

	if !IsMemberInMemberArray(selfMember, initialMembers) {
		return errors.New("the node itself must be inside initial-members")
	}

	initialMembersMap := ToInitialMembersMap(initialMembers)
	if join {
		initialMembersMap = map[uint64]string{}
	}
	return nh.StartOnDiskReplica(initialMembersMap, join, stateMachine, cfg)

}
