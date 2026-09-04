package dragonboat

import (
	"context"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	general_command "deadalus-orch/server/internal/usecase/command/general"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type CommandKind int

const (
	KindEnqueue CommandKind = iota
	KindDequeue
)

type ScheduledTask struct {
	Ctx           context.Context
	RaftNode      *RaftNode
	Cmd           commands.Command
	Timeout       time.Duration
	Logger        zerolog.Logger
	OperationName string
	ResChan       chan TaskResult
}

type TaskResult struct {
	Data []byte
	Err  error
}

type ShardScheduler struct {
	enqueueChan chan ScheduledTask
	dequeueChan chan ScheduledTask
	stopChan    chan struct{}
}

type RaftSchedulerRegistry struct {
	mu         sync.Mutex
	schedulers map[uint64]*ShardScheduler
}

var globalSchedulerRegistry = &RaftSchedulerRegistry{
	schedulers: make(map[uint64]*ShardScheduler),
}

func getOrCreateShardScheduler(shardID uint64) *ShardScheduler {
	globalSchedulerRegistry.mu.Lock()
	defer globalSchedulerRegistry.mu.Unlock()

	if sched, exists := globalSchedulerRegistry.schedulers[shardID]; exists {
		return sched
	}

	sched := &ShardScheduler{
		enqueueChan: make(chan ScheduledTask, 1000),
		dequeueChan: make(chan ScheduledTask, 1000),
		stopChan:    make(chan struct{}),
	}
	globalSchedulerRegistry.schedulers[shardID] = sched

	fmt.Printf("⚡⚡⚡ [RAFT SCHEDULER ACTIVE] Spawning ShardScheduler for ShardID: %d ⚡⚡⚡\n", shardID)
	go sched.run()
	return sched
}

func (s *ShardScheduler) run() {
	for {
		select {
		case <-s.stopChan:
			return
		case task := <-s.enqueueChan:
			s.executeTask(task)
			// Interleave: Try to serve one Dequeue task immediately if pending
			select {
			case deqTask := <-s.dequeueChan:
				s.executeTask(deqTask)
			default:
			}
		case task := <-s.dequeueChan:
			s.executeTask(task)
			// Interleave: Try to serve one Enqueue task immediately if pending
			select {
			case enqTask := <-s.enqueueChan:
				s.executeTask(enqTask)
			default:
			}
		}
	}
}

func (s *ShardScheduler) executeTask(task ScheduledTask) {
	var res TaskResult

	// Use a dedicated background context with timeout for Raft execution so that
	// caller context cancellations (e.g., client stream disconnects or short deadlines)
	// do not cancel the Raft transaction mid-flight or cause context canceled errors.
	writeCtx, writeCancel := context.WithTimeout(context.Background(), task.Timeout)
	defer writeCancel()

	fsmCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.REPOSITORY_COMMAND,
		CMD:  task.Cmd,
	}

	resultChan, err := task.RaftNode.Write(writeCtx, fsmCmd)
	if err != nil {
		res.Err = fmt.Errorf("failed to start %s: %w", task.OperationName, err)
		select {
		case task.ResChan <- res:
		default:
		}
		return
	}

	select {
	case writeResult := <-resultChan:
		if writeResult.Error != nil {
			res.Err = fmt.Errorf("failed to execute %s: %w", task.OperationName, writeResult.Error)
		} else {
			res.Data = writeResult.Result.Data
		}
	case <-writeCtx.Done():
		res.Err = fmt.Errorf("%s operation timed out: %w", task.OperationName, writeCtx.Err())
	}

	select {
	case task.ResChan <- res:
	default:
	}
}

// ExecuteScheduledRepositoryCommand routes repository commands through the Raft Turn Scheduler
// ensuring 1:1 fair interleaving between Enqueue (Publish) and Dequeue/Ack (Consumer) micro-batches.
func ExecuteScheduledRepositoryCommand[T any](
	kind CommandKind,
	raftNode *RaftNode,
	ctx context.Context,
	cmd commands.Command,
	timeout time.Duration,
	logger zerolog.Logger,
	operationName string,
) (T, error) {
	var zero T
	if raftNode == nil {
		return zero, fmt.Errorf("raftNode is nil for %s", operationName)
	}

	sched := getOrCreateShardScheduler(raftNode.ShardID)
	task := ScheduledTask{
		Ctx:           ctx,
		RaftNode:      raftNode,
		Cmd:           cmd,
		Timeout:       timeout,
		Logger:        logger,
		OperationName: operationName,
		ResChan:       make(chan TaskResult, 1),
	}

	if kind == KindEnqueue {
		select {
		case sched.enqueueChan <- task:
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	} else {
		select {
		case sched.dequeueChan <- task:
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	select {
	case res := <-task.ResChan:
		if res.Err != nil {
			return zero, res.Err
		}
		return commands.DecodeCommandResult[T](res.Data)
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}
