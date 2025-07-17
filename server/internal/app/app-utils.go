package app

import "deadalus-orch/server/internal/pkg/config"

func RecommendRTTMillisecond() uint64 {
	shardCount := config.GlobalConfiguration.MaxShards
	switch {
	case shardCount <= 50:
		return 200
	case shardCount <= 100:
		return 250
	case shardCount <= 200:
		return 350
	case shardCount <= 400:
		return 375
	case shardCount <= 800:
		return 450
	case shardCount <= 1600:
		return 500

	default:
		return 300
	}
}
