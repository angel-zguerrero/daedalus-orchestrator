package business_logic

import (
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"
	"runtime"
	"strings"
	"time"

	"deadalus-orch/server/internal/pkg/config"
	node_scheduler "deadalus-orch/server/internal/usecase/command/node-scheduler"
	"deadalus-orch/shared/models"
	"errors"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type NodeSchedulerBO struct {
	Config *common.ServerConfing
}

func NewNodeSchedulerBO(Config *common.ServerConfing) *NodeSchedulerBO {
	return &NodeSchedulerBO{
		Config: Config,
	}
}

func (bo *NodeSchedulerBO) UpsertNodeScheduler(ctx context.Context, code, name string) (models.NodeScheduler, error) {
	nodeScheduler := &models.NodeScheduler{
		ID:   strings.ReplaceAll(uuid.New().String(), "-", ""),
		Name: name,
	}

	createdList, err := bo.BulkUpsertNodeScheduler(ctx, []*models.NodeScheduler{nodeScheduler})
	if err != nil {
		return models.NodeScheduler{}, err
	}
	return createdList[0], nil
}

func (bo *NodeSchedulerBO) BulkUpsertNodeScheduler(ctx context.Context, nodeSchedulers []*models.NodeScheduler) ([]models.NodeScheduler, error) {
	if len(nodeSchedulers) == 0 {
		return nil, errors.New("no nodeSchedulers provided")
	}

	// Asegurar IDs válidos
	for _, t := range nodeSchedulers {
		if t.ID == "" {
			t.ID = strings.ReplaceAll(uuid.New().String(), "-", "")
		}
	}

	// Fetch server resource usage and populate Information field
	for _, t := range nodeSchedulers {
		resourceUsage, err := bo.fetchServerResourceUsage()
		if err != nil {
			bo.Config.Logger.Warn().Err(err).Msgf("Failed to fetch resource usage for server")
			continue
		}
		t.Information = resourceUsage

	}

	upsertNodeSchedulerCommand := &node_scheduler.UpsertNodeSchedulerCommand{
		NodeSchedulers: make([]models.NodeScheduler, len(nodeSchedulers)),
	}
	for i, t := range nodeSchedulers {
		upsertNodeSchedulerCommand.NodeSchedulers[i] = *t
	}

	created, err := dragonboat.ExecuteRepositoryCommand[[]models.NodeScheduler](
		bo.Config.MasterNode,
		ctx,
		upsertNodeSchedulerCommand,
		config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(nodeSchedulers)),
		bo.Config.Logger,
		"bulk upsert nodeSchedulers",
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (bo *NodeSchedulerBO) UpdateRunningStatusNodeScheduler(ctx context.Context, nodeSchedulers []*models.NodeScheduler, runningStatus models.NodeSchedulerRunningStatus) ([]models.NodeScheduler, error) {
	if len(nodeSchedulers) == 0 {
		return nil, errors.New("no nodeSchedulers provided")
	}

	updateRunningStatusNodeSchedulerCommand := &node_scheduler.UpdateRunningStatusNodeSchedulerCommand{
		NodeSchedulers: make([]models.NodeScheduler, len(nodeSchedulers)),
		RunningStatus:  runningStatus,
	}
	for i, t := range nodeSchedulers {
		updateRunningStatusNodeSchedulerCommand.NodeSchedulers[i] = *t
	}

	created, err := dragonboat.ExecuteRepositoryCommand[[]models.NodeScheduler](
		bo.Config.MasterNode,
		ctx,
		updateRunningStatusNodeSchedulerCommand,
		config.GlobalConfiguration.ApiRaftTimeout*time.Duration(len(nodeSchedulers)),
		bo.Config.Logger,
		"update running status command",
	)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (bo *NodeSchedulerBO) GetNodeScheduler(ctx context.Context, nodeSchedulerID string) (models.NodeScheduler, error) {
	findNodeSchedulerCommand := &node_scheduler.FindNodeSchedulerCommand{
		NodeSchedulerID: nodeSchedulerID,
	}

	nodeScheduler, err := dragonboat.ExecuteRepositoryQuery[models.NodeScheduler](
		bo.Config.MasterNode,
		ctx,
		findNodeSchedulerCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find nodeScheduler",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.NodeScheduler{}, errors.New("NodeScheduler not found")
		}
		return models.NodeScheduler{}, fmt.Errorf("find nodeSchedulers command failed: %w", err)
	}

	return nodeScheduler, nil
}

func (bo *NodeSchedulerBO) GetNodeSchedulerByName(ctx context.Context, nodeSchedulerName string) (models.NodeScheduler, error) {
	findNodeSchedulerCommand := &node_scheduler.FindNodeSchedulerCommand{
		NodeSchedulerName: nodeSchedulerName,
	}

	nodeScheduler, err := dragonboat.ExecuteRepositoryQuery[models.NodeScheduler](
		bo.Config.MasterNode,
		ctx,
		findNodeSchedulerCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"find nodeScheduler",
	)
	if err != nil {
		if strings.Contains(err.Error(), "entity not found") {
			return models.NodeScheduler{}, errors.New("NodeScheduler not found")
		}
		return models.NodeScheduler{}, fmt.Errorf("find nodeSchedulers command failed: %w", err)
	}

	return nodeScheduler, nil
}

func (bo *NodeSchedulerBO) GetNodeSchedulers(ctx context.Context, q string, cursor string, pageSize int) (db.FindResult[models.NodeScheduler], error) {
	paginateNodeSchedulersCommand := &node_scheduler.PaginateNodeSchedulersCommand{
		Cursor:   cursor,
		PageSize: pageSize,
		Q:        q,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.NodeScheduler]](
		bo.Config.MasterNode,
		ctx,
		paginateNodeSchedulersCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate nodeSchedulers",
	)
	if err != nil {
		return db.FindResult[models.NodeScheduler]{}, fmt.Errorf("paginate nodeSchedulers failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.NodeScheduler{}
	}

	return findResult, nil
}

func (bo *NodeSchedulerBO) GetNodeSchedulersUsingAssignedTenantNodeIndex(ctx context.Context, q string, cursor string, pageSize int, assignedTenantNodeIndex int) (db.FindResult[models.NodeScheduler], error) {
	paginateNodeSchedulersCommand := &node_scheduler.PaginateNodeSchedulersAssignedTenantNodeIndexCommand{
		Cursor:                  cursor,
		PageSize:                pageSize,
		Q:                       q,
		AssignedTenantNodeIndex: assignedTenantNodeIndex,
	}

	findResult, err := dragonboat.ExecuteRepositoryQuery[db.FindResult[models.NodeScheduler]](
		bo.Config.MasterNode,
		ctx,
		paginateNodeSchedulersCommand,
		config.GlobalConfiguration.ApiRaftTimeout,
		bo.Config.Logger,
		"paginate nodeSchedulers",
	)
	if err != nil {
		return db.FindResult[models.NodeScheduler]{}, fmt.Errorf("paginate nodeSchedulers failed: %w", err)
	}

	if findResult.Entities == nil {
		findResult.Entities = []models.NodeScheduler{}
	}

	return findResult, nil
}

// fetchServerResourceUsage fetches resource usage (CPU, memory, disk) for a given server ID.
func (bo *NodeSchedulerBO) fetchServerResourceUsage() (map[string]string, error) {
	// Directly fetch resource usage information within this method
	resourceUsage := make(map[string]string)

	// Get CPU usage
	cpuPercentages, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CPU usage: %w", err)
	}
	if len(cpuPercentages) > 0 {
		resourceUsage["CPU"] = fmt.Sprintf("%.2f%%", cpuPercentages[0])
	}

	// Get memory usage
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch memory usage: %w", err)
	}
	resourceUsage["Memory"] = fmt.Sprintf("%d/%dMB", virtualMemory.Used/1024/1024, virtualMemory.Total/1024/1024)

	// Get disk usage
	diskUsage, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch disk usage: %w", err)
	}
	resourceUsage["Disk"] = fmt.Sprintf("%d/%dGB", diskUsage.Used/1024/1024/1024, diskUsage.Total/1024/1024/1024)

	// Get OS information
	resourceUsage["OS"] = runtime.GOOS

	return resourceUsage, nil
}
