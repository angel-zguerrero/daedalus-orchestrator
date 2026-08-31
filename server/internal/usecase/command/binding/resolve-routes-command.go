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
	gob.Register(ResolveRoutesCommand{})
	gob.Register(ResolveRoutesResult{})
}

type ResolveRoutesResult struct {
	Targets    []string
	HasDynamic bool
}

type ResolveRoutesCommand struct {
	ExchangeID     string
	ExchangeType   models.ExchangeType
	RoutingKey     string // Or pattern/queue code
	MessageHeaders map[string]string

	CF  string
	CFS string
}

func (cmd *ResolveRoutesCommand) Execute(uow *db.UnitOfWork, now time.Time) command.CommandResult {
	commandResult := &command.CommandResult{}

	rt := db.NewRouteTable(uow.KVStore, cmd.CF, cmd.CFS, "admin_schema")

	var targets []string
	var err error

	switch cmd.ExchangeType {
	case models.Direct:
		targets, err = rt.GetDirectRoutes(cmd.ExchangeID, cmd.RoutingKey, now)

	case models.Fanout:
		targets, err = rt.GetFanoutRoutes(cmd.ExchangeID, now)

	case models.Topic:
		// Attempt to use lazy cache
		cachedTargets, ok, errTopic := rt.GetTopicRouteCache(cmd.ExchangeID, cmd.RoutingKey, now)
		if errTopic != nil {
			err = errTopic
			break
		}
		if ok {
			targets = cachedTargets
			break
		}

		// Cache miss: compute from patterns
		patterns, errTopic := rt.GetTopicPatterns(cmd.ExchangeID, now)
		if errTopic != nil {
			err = errTopic
			break
		}

		// Filter patterns that match
		targetMap := make(map[string]bool)
		for _, p := range patterns {
			if matchesTopicPattern(p.Pattern, cmd.RoutingKey) {
				targetMap[p.QueueID] = true
			}
		}

		for t := range targetMap {
			targets = append(targets, t)
		}

	case models.Headers:
		bindings, errTopic := rt.GetHeadersBindings(cmd.ExchangeID, now)
		if errTopic != nil {
			err = errTopic
			break
		}

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

	HasDynamic, errDyn := rt.HasDynamicRoutes(cmd.ExchangeID, now)
	if errDyn != nil {
		commandResult.Error = errDyn.Error()
		return *commandResult
	}

	if err != nil {
		commandResult.Error = err.Error()
		return *commandResult
	}

	commandResult.Result = ResolveRoutesResult{
		Targets:    targets,
		HasDynamic: HasDynamic,
	}
	return *commandResult
}

// Match topic pattern (AMQP-compliant implementation)
func matchesTopicPattern(pattern, routingKey string) bool {
	if pattern == "" {
		return routingKey == ""
	}
	if pattern == "#" {
		return true // # matches everything
	}

	patternWords := strings.Split(pattern, ".")
	routingWords := strings.Split(routingKey, ".")

	return matchTopicWords(patternWords, routingWords)
}

func matchTopicWords(patternWords, routingWords []string) bool {
	if len(patternWords) == 0 && len(routingWords) == 0 {
		return true
	}
	if len(patternWords) == 0 {
		return false
	}

	currentPattern := patternWords[0]
	remainingPattern := patternWords[1:]

	switch currentPattern {
	case "#":
		if len(remainingPattern) == 0 {
			return true
		}
		for i := 0; i <= len(routingWords); i++ {
			if matchTopicWords(remainingPattern, routingWords[i:]) {
				return true
			}
		}
		return false

	case "*":
		if len(routingWords) == 0 {
			return false
		}
		return matchTopicWords(remainingPattern, routingWords[1:])

	default:
		if len(routingWords) == 0 || currentPattern != routingWords[0] {
			return false
		}
		return matchTopicWords(remainingPattern, routingWords[1:])
	}
}

func matchesHeaders(binding db.HeadersBindingEntry, msgHeaders map[string]string) bool {
	if len(msgHeaders) == 0 {
		return false
	}
	
	matchCount := 0
	for key, expectedVal := range binding.Headers {
		if actualVal, ok := msgHeaders[key]; ok && actualVal == expectedVal {
			matchCount++
		}
	}

	if binding.XMatch == "all" {
		return matchCount == len(binding.Headers)
	}
	return matchCount > 0
}

