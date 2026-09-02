package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"fmt"
	"time"
)

func init() {
	gob.Register(BulkAssertBindingCommand{})
	gob.Register(BulkAssertBindingResult{})
	gob.Register([]models.Binding{})
}

type BulkAssertBindingItem struct {
	NewBindingID          string
	Code                  string
	QueueCode             string
	ExchangeCode          string
	TargetExchangeCode    string
	AlternateExchangeCode string
	VNamespace            string
	RoutingKey            string
	Pattern               string
	XMatch                models.XMatchType
	BindingType           models.BindingType
	TargetExchangeType    models.TargetExchangeType
	Headers               map[string]string
}

type BulkAssertBindingResult struct {
	Results []models.Binding
}

type BulkAssertBindingCommand struct {
	Items []BulkAssertBindingItem
	CF    string
	CFS   string
}

func (cmd *BulkAssertBindingCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	if len(cmd.Items) == 0 {
		commandResult.Result = BulkAssertBindingResult{Results: []models.Binding{}}
		return *commandResult
	}

	idFactory := &db.DeterministicIDGeneratorFactory{}

	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	bindingRepo, err := db.NewBindingRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	routingHeadersRepo, err := db.NewRoutingHeadersRepository(uow, idFactory, cmd.CF, cmd.CFS)
	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	rt := db.NewRouteTable(uow.KVStore, cmd.CF, cmd.CFS, "admin_schema")
	routeBatch := db.NewWriteBatch()

	exchangesCache := make(map[string]*models.Exchange)
	queuesCache := make(map[string]*models.Queue)
	bindingsToCreate := make([]*models.Binding, 0, len(cmd.Items))
	bindingsToUpdate := make([]*models.Binding, 0)
	results := make([]models.Binding, len(cmd.Items))

	topicBulkMap := make(map[string][]db.TopicPatternEntry)
	fanoutBulkMap := make(map[string][]string)
	headersBulkMap := make(map[string][]db.HeadersBindingEntry)

	for i, item := range cmd.Items {
		if item.NewBindingID == "" {
			commandResult.Error = fmt.Sprintf("item %d: NewBindingID is required", i)
			return *commandResult
		}
		if item.Code == "" {
			commandResult.Error = fmt.Sprintf("item %d: Code is required", i)
			return *commandResult
		}
		if item.ExchangeCode == "" {
			commandResult.Error = fmt.Sprintf("item %d: ExchangeCode is required", i)
			return *commandResult
		}

		vns := item.VNamespace
		if vns == "" {
			vns = "default"
		}

		targetExType := item.TargetExchangeType
		if targetExType == "" {
			targetExType = models.TargetExchangeTypeQueue
		}

		bindingType := item.BindingType
		if bindingType == "" {
			bindingType = models.BindingTypeClassic
		}

		exKey := item.ExchangeCode + ":" + vns
		ex, exists := exchangesCache[exKey]
		if !exists {
			var err error
			ex, err = exchangeRepo.GetExchangeByCode(item.ExchangeCode, vns, now)
			if err != nil || ex == nil {
				commandResult.Error = fmt.Sprintf("Exchange %s in vnamespace %s not found", item.ExchangeCode, vns)
				return *commandResult
			}
			exchangesCache[exKey] = ex
		}

		var queueID string
		if targetExType == models.TargetExchangeTypeQueue && bindingType == models.BindingTypeClassic {
			qKey := item.QueueCode + ":" + vns
			q, exists := queuesCache[qKey]
			if !exists {
				var err error
				q, err = queueRepo.GetQueueByCode(item.QueueCode, vns, now)
				if err != nil || q == nil {
					commandResult.Error = fmt.Sprintf("Queue %s in vnamespace %s not found", item.QueueCode, vns)
					return *commandResult
				}
				queuesCache[qKey] = q
			}
			queueID = q.ID
		}

		var targetExID string
		if targetExType == models.TargetExchangeTypeExchange {
			tExKey := item.TargetExchangeCode + ":" + vns
			tex, exists := exchangesCache[tExKey]
			if !exists {
				var err error
				tex, err = exchangeRepo.GetExchangeByCode(item.TargetExchangeCode, vns, now)
				if err != nil || tex == nil {
					commandResult.Error = fmt.Sprintf("Target exchange %s in vnamespace %s not found", item.TargetExchangeCode, vns)
					return *commandResult
				}
				exchangesCache[tExKey] = tex
			}
			targetExID = tex.ID
		}

		existingBinding, err := bindingRepo.GetBindingByCode(item.Code, vns, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed checking binding %s: %s", item.Code, err.Error())
			return *commandResult
		}

		var b *models.Binding
		if existingBinding != nil {
			oldTarget := ""
			if existingBinding.TargetExchangeType == models.TargetExchangeTypeExchange && existingBinding.TargetExchangeID != "" {
				oldTarget = "e:" + existingBinding.TargetExchangeID
			} else if existingBinding.QueueID != "" {
				oldTarget = "q:" + existingBinding.QueueID
			}

			b = existingBinding
			b.ExchangeID = ex.ID
			b.ExchangeCode = item.ExchangeCode
			b.QueueID = queueID
			b.QueueCode = item.QueueCode
			b.TargetExchangeID = targetExID
			b.TargetExchangeCode = item.TargetExchangeCode
			b.AlternateExchangeCode = item.AlternateExchangeCode
			b.RoutingKey = item.RoutingKey
			b.Pattern = item.Pattern
			b.XMatch = item.XMatch
			b.BindingType = bindingType
			b.TargetExchangeType = targetExType
			b.Headers = item.Headers
			b.UpdatedAt = now
			bindingsToUpdate = append(bindingsToUpdate, b)

			newTarget := ""
			if b.TargetExchangeType == models.TargetExchangeTypeExchange && b.TargetExchangeID != "" {
				newTarget = "e:" + b.TargetExchangeID
			} else if b.QueueID != "" {
				newTarget = "q:" + b.QueueID
			}

			if oldTarget != "" && oldTarget != newTarget {
				switch ex.Type {
				case models.Direct:
					if existingBinding.RoutingKey != "" {
						_ = rt.RemoveDirectRoute(routeBatch, ex.ID, existingBinding.RoutingKey, oldTarget, now)
					}
				case models.Fanout:
					_ = rt.RemoveFanoutRoute(routeBatch, ex.ID, oldTarget, now)
				}
			}
		} else {
			b = &models.Binding{
				ID:                    item.NewBindingID,
				Code:                  item.Code,
				ExchangeID:            ex.ID,
				ExchangeCode:          item.ExchangeCode,
				QueueID:               queueID,
				QueueCode:             item.QueueCode,
				TargetExchangeID:      targetExID,
				TargetExchangeCode:    item.TargetExchangeCode,
				AlternateExchangeCode: item.AlternateExchangeCode,
				VNamespace:            vns,
				RoutingKey:            item.RoutingKey,
				Pattern:               item.Pattern,
				XMatch:                item.XMatch,
				BindingType:           bindingType,
				TargetExchangeType:    targetExType,
				Headers:               item.Headers,
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			bindingsToCreate = append(bindingsToCreate, b)
		}

		// Update route table entry
		var target string
		if b.TargetExchangeType == models.TargetExchangeTypeExchange && b.TargetExchangeID != "" {
			target = "e:" + b.TargetExchangeID
		} else if b.QueueID != "" {
			target = "q:" + b.QueueID
		}

		if target != "" {
			switch ex.Type {
			case models.Direct:
				if b.BindingType == models.BindingTypeClassic && b.RoutingKey != "" {
					_ = rt.AddDirectRoute(routeBatch, ex.ID, b.RoutingKey, target, now)
				} else if b.BindingType == models.BindingTypeDynamic {
					_ = rt.AddDynamicRoute(routeBatch, ex.ID, now)
				}
			case models.Fanout:
				if b.BindingType == models.BindingTypeClassic {
					fanoutBulkMap[ex.ID] = append(fanoutBulkMap[ex.ID], target)
				} else if b.BindingType == models.BindingTypeDynamic {
					_ = rt.AddDynamicRoute(routeBatch, ex.ID, now)
				}
			case models.Topic:
				if b.BindingType == models.BindingTypeClassic && b.Pattern != "" {
					topicBulkMap[ex.ID] = append(topicBulkMap[ex.ID], db.TopicPatternEntry{
						Pattern:   b.Pattern,
						QueueID:   target,
						BindingID: b.ID,
					})
				} else if b.BindingType == models.BindingTypeDynamic {
					_ = rt.AddDynamicRoute(routeBatch, ex.ID, now)
				}
			case models.Headers:
				if b.BindingType == models.BindingTypeClassic {
					headersMap := item.Headers
					if headersMap == nil {
						headersMap = map[string]string{}
					}
					xmatch := string(b.XMatch)
					if xmatch == "" {
						xmatch = string(models.XMatchTypeAll)
					}
					headersBulkMap[ex.ID] = append(headersBulkMap[ex.ID], db.HeadersBindingEntry{
						BindingID: b.ID,
						Headers:   headersMap,
						XMatch:    xmatch,
						QueueID:   target,
					})
				} else if b.BindingType == models.BindingTypeDynamic {
					_ = rt.AddDynamicRoute(routeBatch, ex.ID, now)
				}
			}
		}

		results[i] = *b
	}

	for exID, entries := range topicBulkMap {
		if err := rt.AddTopicPatternsBulk(routeBatch, exID, entries, now); err != nil {
			commandResult.Error = fmt.Sprintf("failed adding topic patterns bulk: %s", err.Error())
			return *commandResult
		}
	}
	for exID, targets := range fanoutBulkMap {
		if err := rt.AddFanoutRoutesBulk(routeBatch, exID, targets, now); err != nil {
			commandResult.Error = fmt.Sprintf("failed adding fanout routes bulk: %s", err.Error())
			return *commandResult
		}
	}
	for exID, entries := range headersBulkMap {
		if err := rt.AddHeadersBindingsBulk(routeBatch, exID, entries, now); err != nil {
			commandResult.Error = fmt.Sprintf("failed adding headers bindings bulk: %s", err.Error())
			return *commandResult
		}
	}

	if len(bindingsToCreate) > 0 {
		_, err := bindingRepo.BulkCreate(bindingsToCreate, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to bulk create bindings: %s", err.Error())
			return *commandResult
		}
	}

	if len(bindingsToUpdate) > 0 {
		_, err := bindingRepo.BulkUpdate(bindingsToUpdate, now)
		if err != nil {
			commandResult.Error = fmt.Sprintf("failed to bulk update bindings: %s", err.Error())
			return *commandResult
		}
	}

	if routeBatch.Count() > 0 {
		if err := uow.KVStore.Write(routeBatch); err != nil {
			commandResult.Error = fmt.Sprintf("failed to write route table batch: %s", err.Error())
			return *commandResult
		}
	}

	for _, b := range results {
		if b.Headers != nil && len(b.Headers) > 0 {
			for k, v := range b.Headers {
				rh := &models.RoutingHeader{
					ID:         idFactory.GenerateID(),
					BindingID:  b.ID,
					Key:        k,
					Value:      v,
					HeaderType: models.HeaderTypeBinding,
					VNamespace: b.VNamespace,
					CreatedAt:  now,
					UpdatedAt:  now,
				}
				routingHeadersRepo.Create(rh, now)
			}
		}
	}

	commandResult.Result = BulkAssertBindingResult{Results: results}
	return *commandResult
}
