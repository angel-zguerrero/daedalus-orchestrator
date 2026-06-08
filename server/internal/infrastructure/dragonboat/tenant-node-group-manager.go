package dragonboat

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/pkg/config"
	"sync"

	"github.com/lni/dragonboat/v4"
)

func StartTentantNodes(
	replicaID uint64,
	selfMember Member,
	join bool,
	roles []NodeRole,
	pathProvider db.PathProvider,
	sharedDBProvider *db.SharedDBProvider,
	initialMembers []Member,
	NH *dragonboat.NodeHost,
) ([]*RaftNode, error) {
	MaxShards := config.GlobalConfiguration.MaxShards

	var (
		tenantNodes []*RaftNode
		mu          sync.Mutex
		wg          sync.WaitGroup
		errOnce     sync.Once
		firstErr    error
		semaphore   = make(chan struct{}, 20)
	)

	for shardID := 0; shardID < MaxShards; shardID++ {
		wg.Add(1)

		go func(shardID int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			node, err := InitTenantNode(
				uint64(shardID+MasterShardID)+1,
				replicaID,
				selfMember,
				initialMembers,
				join,
				roles,
				NH,
				pathProvider,
				sharedDBProvider,
			)
			if err != nil {
				errOnce.Do(func() {
					firstErr = err
				})
				return
			}

			mu.Lock()
			tenantNodes = append(tenantNodes, node)
			mu.Unlock()
		}(shardID)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return tenantNodes, nil
}
