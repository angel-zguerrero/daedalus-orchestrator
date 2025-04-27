package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"strconv"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
)

func Init(ShardID uint64, ReplicaID uint64, port string) {

	cfg := config.Config{
		ReplicaID:          ReplicaID,
		ShardID:            ShardID,
		CheckQuorum:        true,
		ElectionRTT:        10,
		HeartbeatRTT:       1,
		SnapshotEntries:    1000,
		CompactionOverhead: 500,
	}

	stateMachine := func(clusterID uint64, nodeID uint64) statemachine.IOnDiskStateMachine {
		return NewKVStateMachine(clusterID, nodeID)
	}

	base_path, err := db.DefaultPathProvider{}.GetDatabasePath()
	if err != nil {
		panic(err)
	}
	nh, err := dragonboat.NewNodeHost(config.NodeHostConfig{
		WALDir:         base_path + "/wal/" + strconv.FormatUint(ReplicaID, 10) + "/" + port,
		NodeHostDir:    base_path + "/node/" + strconv.FormatUint(ReplicaID, 10) + "/" + port,
		RTTMillisecond: 200,
		RaftAddress:    "localhost:" + port,
	})
	if err != nil {
		panic(err)
	}

	initialMembers := map[uint64]string{
		1: "localhost:3435",
		2: "localhost:3436",
	}
	err = nh.StartOnDiskReplica(initialMembers, false, stateMachine, cfg)

}
