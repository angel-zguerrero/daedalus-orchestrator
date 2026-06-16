package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"deadalus-orch/server/internal/pkg/config"
	metrics_command "deadalus-orch/server/internal/usecase/command/metrics"
	"deadalus-orch/shared/models"
	"fmt"
)

type TSDBMetricsBO struct {
	Config *common.ServerConfing
}

func NewTSDBMetricsBO(Config *common.ServerConfing) *TSDBMetricsBO {
	return &TSDBMetricsBO{
		Config: Config,
	}
}

func (bo *TSDBMetricsBO) QueryMetrics(ctx context.Context, tenantCode, queueCode, vnamespace string, resolution int, startTime, endTime int64) (models.MetricsQueryResult, error) {
	// Need to query the node where the tenant data lives
	tenantBO := NewTenantBO(bo.Config)
	tenant, tenantNode, _, err := tenantBO.GetTenant(ctx, tenantCode)
	if err != nil {
		return models.MetricsQueryResult{}, fmt.Errorf("failed to get tenant %s: %w", tenantCode, err)
	}

	if tenantNode == nil {
		return models.MetricsQueryResult{}, fmt.Errorf("no node found for tenant %s", tenantCode)
	}

	cmd := &metrics_command.QueryMetricsRangeCommand{
		TenantCode: tenant.Code,
		QueueCode:  queueCode,
		VNamespace: vnamespace,
		Resolution: resolution,
		FromUnix:   startTime,
		ToUnix:     endTime,
		Limit:      1000,
		Scope:      "queue", // we default to queue scope if queueCode is passed
	}

	if queueCode == "" {
		cmd.Scope = "tenant"
	}

	result, err := dragonboat.ExecuteRepositoryQuery[models.MetricsQueryResult](
		tenantNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"query metrics",
	)

	if err != nil {
		return models.MetricsQueryResult{}, fmt.Errorf("query metrics failed: %w", err)
	}

	return result, nil
}

func (bo *TSDBMetricsBO) QueryGlobalMetrics(ctx context.Context, resolution int, startTime, endTime int64) (models.MetricsQueryResult, error) {
	cmd := &metrics_command.QueryMetricsRangeCommand{
		Resolution: resolution,
		FromUnix:   startTime,
		ToUnix:     endTime,
		Limit:      1000,
		Scope:      "global",
	}

	// Gather metrics from all Tenant Shard nodes
	type datapointAgg struct {
		Published    uint64
		Delivered    uint64
		Acked        uint64
		Failed       uint64
		Pending      uint64
		InProcess    uint64
		MsgPerSecond float64
		LatencySumMs float64
		LatencyCount uint64
		MaxLatencyMs uint64
	}

	aggregatedMap := make(map[int64]*datapointAgg)

	// In the distributed metrics relay architecture, all TSDB metrics from the 
	// Tenant Shards are asynchronously relayed to the Master Node. 
	// Therefore, the Master Node acts as the single source of truth for global metrics.
	// We only need to query it once!
	result, err := dragonboat.ExecuteRepositoryQuery[models.MetricsQueryResult](
		bo.Config.MasterNode,
		ctx,
		cmd,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"query global metrics from master node",
	)
	
	if err == nil {
		for _, dp := range result.Datapoints {
			agg, exists := aggregatedMap[dp.Timestamp]
			if !exists {
				agg = &datapointAgg{}
				aggregatedMap[dp.Timestamp] = agg
			}

			agg.Published += dp.Published
			agg.Delivered += dp.Delivered
			agg.Acked += dp.Acked
			agg.Failed += dp.Failed
			agg.Pending += dp.Pending
			agg.InProcess += dp.InProcess
			agg.MsgPerSecond += dp.MsgPerSecond
			agg.LatencySumMs += dp.AvgLatencyMs // Approximate sum of averages
			agg.LatencyCount += 1
			if dp.MaxLatencyMs > agg.MaxLatencyMs {
				agg.MaxLatencyMs = dp.MaxLatencyMs
			}
		}
	} else {
		bo.Config.Logger.Error().Err(err).Msg("Failed to query global metrics from master node")
	}

	// Prepare final result
	finalDatapoints := make([]models.MetricsDatapoint, 0)
	
	// We want them ordered by timestamp, let's collect them first
	var minTs, maxTs int64
	minTs = endTime
	maxTs = startTime
	
	for ts := range aggregatedMap {
		if ts < minTs {
			minTs = ts
		}
		if ts > maxTs {
			maxTs = ts
		}
	}
	
	// Ensure we generate the array linearly
	normalizedStartTime := (startTime / int64(resolution)) * int64(resolution)
	normalizedEndTime := (endTime / int64(resolution)) * int64(resolution)
	
	for ts := normalizedStartTime; ts <= normalizedEndTime; ts += int64(resolution) {
		agg, exists := aggregatedMap[ts]
		if exists {
			avgLatency := float64(0)
			if agg.LatencyCount > 0 {
				avgLatency = agg.LatencySumMs / float64(agg.LatencyCount)
			}
			
			finalDatapoints = append(finalDatapoints, models.MetricsDatapoint{
				Timestamp:    ts,
				Published:    agg.Published,
				Delivered:    agg.Delivered,
				Acked:        agg.Acked,
				Failed:       agg.Failed,
				Pending:      agg.Pending,
				InProcess:    agg.InProcess,
				MsgPerSecond: agg.MsgPerSecond,
				AvgLatencyMs: avgLatency,
				MaxLatencyMs: agg.MaxLatencyMs,
			})
		}
	}

	return models.MetricsQueryResult{
		Resolution: resolution,
		Datapoints: finalDatapoints,
	}, nil
}
