package app

import "deadalus-orch/server/internal/pkg/config"

func RecommendRTTMillisecond() uint64 {
	shardCount := config.GlobalConfiguration.MaxShards
	switch {
	case shardCount <= 50:
		return 100 // Increased from 50
	case shardCount <= 100:
		return 150 // Increased from 75
	case shardCount <= 200:
		return 200 // Increased from 100
	case shardCount <= 400:
		return 250 // Increased from 150
	case shardCount <= 800:
		return 300 // Increased from 200
	case shardCount <= 1600:
		return 350 // Increased from 250

	default:
		return 200
	}
}
