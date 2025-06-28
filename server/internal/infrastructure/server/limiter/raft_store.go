package ratelimit

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/utils"
	commands "deadalus-orch/server/internal/usecase/command"
	"deadalus-orch/shared/models"
	"encoding/gob"
	"time"

	"github.com/ulule/limiter/v3"
)

type RaftStore struct {
	node     *dragonboat.RaftNode
	keyspace string
	ttl      time.Duration
}

func NewRaftStore(node *dragonboat.RaftNode, keyspace string, ttl time.Duration) *RaftStore {
	return &RaftStore{
		node:     node,
		keyspace: keyspace,
		ttl:      ttl,
	}
}

func (s *RaftStore) fullKey(key string) string {
	return s.keyspace + ":" + key
}

func (s *RaftStore) Increment(ctx context.Context, key string, quantity int64, rate limiter.Rate) (limiter.Context, error) {
	fullKey := s.fullKey(key)
	now := utils.GetNowInInt()

	readCmd := commands.Query_Command{
		Command: commands.RK_Command{
			Key:              fullKey,
			ColumnFamilyName: db.MasterEventFC,
		},
		Now: now,
	}

	resp, err := s.node.Read(ctx, readCmd)
	if err != nil {
		return limiter.Context{}, err
	}

	var state *models.RateLimitState
	if resp == nil {
		// No key exists yet
		state = &models.RateLimitState{
			Limit:     rate.Limit,
			Remaining: rate.Limit - quantity,
			Reset:     int64(s.ttl.Seconds()),
			Reached:   quantity > rate.Limit,
		}
	} else {
		// Decode gob
		buf := bytes.NewBuffer(resp.([]byte))
		dec := gob.NewDecoder(buf)
		state = &models.RateLimitState{}
		if err := dec.Decode(state); err != nil {
			return limiter.Context{}, err
		}

		if state.Remaining > 0 {
			if quantity >= state.Remaining {
				state.Remaining = 0
				state.Reached = true
			} else {
				state.Remaining -= quantity
				state.Reached = false
			}
		} else {
			state.Reached = true
		}

		state.Reset = int64(s.ttl.Seconds())
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return limiter.Context{}, err
	}

	writeCmd := commands.FSM_Command{
		Now:  now,
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              fullKey,
				Value:            buf.Bytes(),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              int(s.ttl.Seconds()),
				Op:               commands.PutOpTTL,
			},
		},
	}

	if _, err := s.node.Write(ctx, writeCmd); err != nil {
		return limiter.Context{}, err
	}

	return limiter.Context{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Reset:     time.Now().Add(s.ttl).Unix(),
		Reached:   state.Reached,
	}, nil
}

func (s *RaftStore) Peek(ctx context.Context, key string, rate limiter.Rate) (limiter.Context, error) {
	fullKey := s.fullKey(key)

	readCmd := commands.Query_Command{
		Command: commands.RK_Command{
			Key:              fullKey,
			ColumnFamilyName: db.MasterEventFC,
		},
		Now: utils.GetNowInInt(),
	}

	resp, err := s.node.Read(ctx, readCmd)
	if err != nil {
		return limiter.Context{}, err
	}

	if resp == nil {
		return limiter.Context{
			Limit:     rate.Limit,
			Remaining: rate.Limit,
			Reset:     time.Now().Add(s.ttl).Unix(),
			Reached:   false,
		}, nil
	}

	buf := bytes.NewBuffer(resp.([]byte))
	dec := gob.NewDecoder(buf)
	state := &models.RateLimitState{}
	if err := dec.Decode(state); err != nil {
		return limiter.Context{}, err
	}

	return limiter.Context{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Reset:     time.Now().Add(s.ttl).Unix(),
		Reached:   state.Remaining <= 0,
	}, nil
}

func (s *RaftStore) Set(ctx context.Context, key string, c limiter.Context) error {
	fullKey := s.fullKey(key)

	state := &models.RateLimitState{
		Limit:     c.Limit,
		Remaining: c.Remaining,
		Reset:     int64(s.ttl.Seconds()),
		Reached:   c.Reached,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return err
	}

	writeCmd := commands.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: commands.RW,
		CMD: commands.RWK_Command{
			Op: commands.Write,
			CMD: commands.WK_Command{
				Key:              fullKey,
				Value:            buf.Bytes(),
				ColumnFamilyName: db.MasterEventFC,
				TTL:              int(s.ttl.Seconds()),
				Op:               commands.PutOpTTL,
			},
		},
	}

	_, err := s.node.Write(ctx, writeCmd)
	return err
}
