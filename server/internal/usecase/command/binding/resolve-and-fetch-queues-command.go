package binding

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"strings"
	"time"
)

func init() {
	gob.Register(ResolveAndFetchQueuesCommand{})
	gob.Register(ResolveAndFetchQueuesResult{})
}

// ResolveAndFetchQueuesResult is the combined result:
// exchange lookup + route resolution + queue hydration in a single Raft read.
type ResolveAndFetchQueuesResult struct {
	// Queues that were resolved directly (q: targets)
	Queues []models.Queue
	// Exchange-to-exchange targets that need recursive resolution (e: targets).
	// Each entry is {ExchangeID, ExchangeCode} so the caller can recurse without
	// another lookup.
	ExchangeTargets []ExchangeTarget
	// HasDynamic signals the caller must also do a dynamic-queue lookup.
	HasDynamic   bool
	ExchangeType models.ExchangeType
}

type ExchangeTarget struct {
	ID   string
	Code string
	Type models.ExchangeType
}

// ResolveAndFetchQueuesCommand combines three former Raft queries into one:
//  1. Find Exchange by Code       (was FindExchangeCommand)
//  2. Resolve routes via RouteTable (was ResolveRoutesCommand)
//  3. Fetch Queue objects by ID     (was FindQueueByIDsCommand)
type ResolveAndFetchQueuesCommand struct {
	ExchangeCode   string
	RoutingKey     string
	MessageHeaders map[string]string
	VNamespace     string
	CF             string
	CFS            string

	// Optional: if set, skip the exchange lookup (caller already has this from cache).
	ExchangeID   string
	ExchangeType models.ExchangeType
}

func (cmd *ResolveAndFetchQueuesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}
	idFactory := &db.DeterministicIDGeneratorFactory{}

	var exchangeID string
	var exchangeType models.ExchangeType

	if cmd.ExchangeID != "" {
		// Fast path: caller provided cached exchange data, skip KV lookup
		exchangeID = cmd.ExchangeID
		exchangeType = cmd.ExchangeType
	} else {
		// Slow path: look up exchange by code
		exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}

		exchange, err := exchangeRepo.GetExchangeByCode(cmd.ExchangeCode, cmd.VNamespace, now)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		if exchange == nil {
			commandResult.Error = "exchange not found"
			return *commandResult
		}
		exchangeID = exchange.ID
		exchangeType = exchange.Type
	}

	// ── Step 2: Resolve routes via RouteTable ────────────────────────────────
	rt := db.NewRouteTable(uow.KVStore, cmd.CF, cmd.CFS, "admin_schema")

	var targets []string
	var err error

	switch exchangeType {
	case models.Direct:
		targets, err = rt.GetDirectRoutes(exchangeID, cmd.RoutingKey, now)

	case models.Fanout:
		targets, err = rt.GetFanoutRoutes(exchangeID, now)

	case models.Topic:
		cachedTargets, ok, errT := rt.GetTopicRouteCache(exchangeID, cmd.RoutingKey, now)
		if errT != nil {
			err = errT
		} else if ok {
			targets = cachedTargets
		} else {
			patterns, errT := rt.GetTopicPatterns(exchangeID, now)
			if errT != nil {
				err = errT
			} else {
				targetMap := make(map[string]bool)
				for _, p := range patterns {
					if matchesTopicPattern(p.Pattern, cmd.RoutingKey) {
						targetMap[p.QueueID] = true
					}
				}
				for t := range targetMap {
					targets = append(targets, t)
				}
			}
		}

	case models.Headers:
		bindings, errH := rt.GetHeadersBindings(exchangeID, now)
		if errH != nil {
			err = errH
		} else {
			targetMap := make(map[string]bool)
			for _, b := range bindings {
				if matchesHeaders(b, cmd.MessageHeaders) {
					targetMap[b.QueueID] = true
				}
			}
			for t := range targetMap {
				targets = append(targets, t)
			}
		}
	}

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	hasDynamic, errDyn := rt.HasDynamicRoutes(exchangeID, now)
	if errDyn != nil {
		commandResult.Error = errDyn.Error()
		return *commandResult
	}

	// ── Step 3: Separate queue IDs vs exchange IDs, then hydrate queues ──────
	var queueIDs []string
	var exchangeTargetIDs []string

	for _, target := range targets {
		if strings.HasPrefix(target, "q:") {
			queueIDs = append(queueIDs, strings.TrimPrefix(target, "q:"))
		} else if strings.HasPrefix(target, "e:") {
			exchangeTargetIDs = append(exchangeTargetIDs, strings.TrimPrefix(target, "e:"))
		}
	}

	// Hydrate queues in one pass
	var queues []models.Queue
	if len(queueIDs) > 0 {
		queueRepo, err := db.NewQueueRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		for _, qID := range queueIDs {
			q, err := queueRepo.GetQueueById(qID, now)
			if err != nil || q == nil {
				continue
			}
			queues = append(queues, *q)
		}
	}

	// Hydrate exchange targets (just code + type, no headers needed for routing)
	var exchangeTargets []ExchangeTarget
	if len(exchangeTargetIDs) > 0 {
		exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cmd.CF, cmd.CFS)
		if err != nil {
			commandResult.Error = err.Error()
			return *commandResult
		}
		for _, exID := range exchangeTargetIDs {
			ex, err := exchangeRepo.GetExchangeById(exID, now)
			if err != nil || ex == nil {
				continue
			}
			exchangeTargets = append(exchangeTargets, ExchangeTarget{
				ID:   ex.ID,
				Code: ex.Code,
				Type: ex.Type,
			})
		}
	}

	commandResult.Result = ResolveAndFetchQueuesResult{
		Queues:          queues,
		ExchangeTargets: exchangeTargets,
		HasDynamic:      hasDynamic,
		ExchangeType:    exchangeType,
	}
	return *commandResult
}
