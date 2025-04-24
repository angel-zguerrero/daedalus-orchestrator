package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"

	"github.com/lni/dragonboat/v4"
	"github.com/lni/dragonboat/v4/config"
	"github.com/lni/dragonboat/v4/statemachine"
)

func Init(store db.KVStore, ShardID uint64, ReplicaID uint64, port string) {

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
		return NewOnDiskKVStateMachine(store)
	}

	nh, err := dragonboat.NewNodeHost(config.NodeHostConfig{
		WALDir:         "./data/wal/" + port,
		NodeHostDir:    "./data/node/" + port,
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
