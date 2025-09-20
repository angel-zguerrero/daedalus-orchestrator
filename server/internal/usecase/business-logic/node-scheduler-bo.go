package business_logic

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/infrastructure/server/common"
	"fmt"
	"runtime"
	"strings"
	"time"

	"deadalus-orch/server/internal/pkg/config"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	node_scheduler "deadalus-orch/server/internal/usecase/command/node-scheduler"
	"deadalus-orch/shared/models"
	"encoding/gob"
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

func (bo *NodeSchedulerBO) GetNodeScheduler(ctx context.Context, nodeSchedulerID string) (models.NodeScheduler, error) {
	findNodeSchedulerCommand := &node_scheduler.FindNodeSchedulerCommand{
		NodeSchedulerID: nodeSchedulerID,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: findNodeSchedulerCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		if strings.Contains(err.Error(), "cannot encode nil pointer of type") {
			return models.NodeScheduler{}, errors.New("NodeScheduler not found")
		}
		bo.Config.Logger.Error().Err(err).Msg("Find nodeSchedulers command failed")
		return models.NodeScheduler{}, errors.New("Find nodeSchedulers command failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Find nodeSchedulers command failed")
		return models.NodeScheduler{}, errors.New("Find nodeSchedulers command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find nodeSchedulers command failed")
		return models.NodeScheduler{}, errors.New("Find nodeSchedulers command failed")
	}

	if parsedResult.Result == nil {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Find nodeSchedulers command failed")
		return models.NodeScheduler{}, errors.New("NodeScheduler not found")
	}

	nodeScheduler := parsedResult.Result.(models.NodeScheduler)

	return nodeScheduler, nil
}

func (bo *NodeSchedulerBO) GetNodeSchedulers(ctx context.Context, q string, cursor string, pageSize int) (db.FindResult[models.NodeScheduler], error) {
	paginateNodeSchedulersCommand := &node_scheduler.PaginateNodeSchedulersCommand{
		Cursor:   cursor,
		PageSize: pageSize,
		Q:        q,
	}

	queryCommand := &general_command.Query_Command{
		Command: &general_command.Repository_Command{
			CMD: paginateNodeSchedulersCommand,
		},
		Now: time.Now().UnixNano(),
	}

	readCtx, cancel := context.WithTimeout(ctx, config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	result, err := bo.Config.MasterNode.Read(readCtx, *queryCommand)
	if err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate nodeSchedulers command failed")
		return db.FindResult[models.NodeScheduler]{}, errors.New("Login failed: " + err.Error())
	}

	buf := bytes.NewBuffer(result.([]byte))
	dec := gob.NewDecoder(buf)
	parsedResult := &commands.CommandResult{}
	if err := dec.Decode(parsedResult); err != nil {
		bo.Config.Logger.Error().Err(err).Msg("Paginate nodeSchedulers command failed")
		return db.FindResult[models.NodeScheduler]{}, errors.New("Paginate nodeSchedulers command failed")
	}

	if parsedResult.Error != "" {
		bo.Config.Logger.Error().Err(err).Str("error", parsedResult.Error).Msg("Paginate nodeSchedulers command failed")
		return db.FindResult[models.NodeScheduler]{}, errors.New("Paginate nodeSchedulers command failed")
	}

	findResult := parsedResult.Result.(db.FindResult[models.NodeScheduler])
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
