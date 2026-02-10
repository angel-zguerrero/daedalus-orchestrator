package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	models "deadalus-orch/shared/models"
)

type NodeSchedulerRepository struct {
	*Repository[models.NodeScheduler]
}

func NewNodeSchedulerRepository(uow *UnitOfWork, factory IDGeneratorFactory) (*NodeSchedulerRepository, error) {
	if uow == nil {
		return nil, fmt.Errorf("UnitOfWork is required")
	}
	repo, err := GetRepository[models.NodeScheduler](uow, MasterEventFC, MasterEventFCSector, "admin_event_schema", factory)
	if err != nil {
		return nil, err
	}
	return &NodeSchedulerRepository{Repository: repo}, nil
}

func (r *NodeSchedulerRepository) CreateNodeScheduler(input *models.NodeScheduler, now time.Time) (string, error) {
	input.CreatedAt = now
	input.UpdatedAt = now
	return r.Create(input, now)
}

func (r *NodeSchedulerRepository) UpdateNodeScheduler(input *models.NodeScheduler, now time.Time) (bool, error) {
	input.UpdatedAt = now
	return r.Update(input, now)
}

func (r *NodeSchedulerRepository) GetNodeSchedulerByName(name string, now time.Time) (*models.NodeScheduler, error) {
	return r.FindByField("Name", name, now)
}
func (r *NodeSchedulerRepository) GetNodeSchedulerById(id string, now time.Time) (*models.NodeScheduler, error) {
	return r.FindByField("ID", id, now)
}

func (r *NodeSchedulerRepository) Paginate(q string, balancingId string, connectionStatus models.ConnectionStatus, AssignedTenantNodeIndex int, pageSize int, cursor string, now time.Time) (*FindResult[models.NodeScheduler], error) {
	var conditions []string

	// Add name search condition if q is provided
	if q != "" {
		conditions = append(conditions, "Name LIKE *"+q+"*")
	}

	// Add balancingId	 filter condition if vNamespace is provided
	if balancingId != "" {
		conditions = append(conditions, "BalancingId = "+balancingId)
	}

	// Add connectionStatus filter condition if vNamespace is provided
	if connectionStatus != "" {
		conditions = append(conditions, "ConnectionStatus = "+string(connectionStatus))
	}

	// Add AssignedTenantNodeIndex filter condition if vNamespace is provided
	if AssignedTenantNodeIndex > -1 {
		conditions = append(conditions, "AssignedTenantNodeIndex = "+strconv.Itoa(AssignedTenantNodeIndex))
	}

	// If no conditions but we got here, use the workaround
	if len(conditions) == 0 {
		return r.Find("ID != 0", pageSize, cursor, now) // ID != 0 Workaround
	} else {
		return r.Find(strings.Join(conditions, " & "), pageSize, cursor, now) // ID != 0 Workaround
	}
}

func (r *NodeSchedulerRepository) DeleteNodeSchedulerById(id string, now time.Time) (bool, error) {
	return r.Delete(id, now)
}
