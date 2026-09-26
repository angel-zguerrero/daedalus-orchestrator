package activity

import (
	"context"
	"strings"
	"sync"
)

// ActivityExecutor defines the interface for native activity execution in WORKER_ACTIVITIES.
type ActivityExecutor interface {
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

// Registry manages the available native activity executors.
type Registry struct {
	mu        sync.RWMutex
	executors map[string]ActivityExecutor
}

var defaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	r := &Registry{
		executors: make(map[string]ActivityExecutor),
	}
	r.RegisterDefaults()
	return r
}

func GetRegistry() *Registry {
	return defaultRegistry
}

func (r *Registry) Register(activityType string, executor ActivityExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(activityType))
	r.executors[key] = executor
}

func (r *Registry) Get(activityType string) (ActivityExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := strings.ToLower(strings.TrimSpace(activityType))

	if exec, found := r.executors[key]; found {
		return exec, true
	}

	// Secondary match for partial aliases (e.g. contains "http" or "redis")
	for k, exec := range r.executors {
		if strings.Contains(key, k) {
			return exec, true
		}
	}

	return nil, false
}

func (r *Registry) RegisterDefaults() {
	httpExec := &HTTPExecutor{}
	redisExec := &RedisExecutor{}
	logExec := &LogExecutor{}
	scriptExec := &ScriptExecutor{}

	// HTTP-based services (HTTP REST)
	r.executors["io.camunda.connectors.httpjson.v1"] = httpExec
	r.executors["io.camunda:http-json:1"] = httpExec
	r.executors["http-json"] = httpExec
	r.executors["http"] = httpExec
	r.executors["rest"] = httpExec

	// Redis
	r.executors["io.camunda.connectors.redis.v1"] = redisExec
	r.executors["io.camunda:connector-redis:1"] = redisExec
	r.executors["io.camunda:connector-redis"] = redisExec
	r.executors["redis"] = redisExec

	// JavaScript Script Task
	r.executors["io.camunda.connectors.scripttask.v1"] = scriptExec
	r.executors["scripttask"] = scriptExec
	r.executors["script-task"] = scriptExec
	r.executors["javascript"] = scriptExec
	r.executors["script"] = scriptExec

	// Log / Debug & Standard BPMN Task Elements
	r.executors["io.camunda.connectors.logtask.v1"] = logExec
	r.executors["logtask"] = logExec
	r.executors["log"] = logExec
	r.executors["task"] = logExec
	r.executors["servicetask"] = logExec
	r.executors["service-task"] = logExec
	r.executors["usertask"] = logExec
	r.executors["user-task"] = logExec
	r.executors["sendtask"] = logExec
	r.executors["send-task"] = logExec
	r.executors["receivetask"] = logExec
	r.executors["receive-task"] = logExec
	r.executors["manualtask"] = logExec
	r.executors["manual-task"] = logExec
	r.executors["businessruletask"] = logExec
	r.executors["business-rule-task"] = logExec
}
