package ratelimit

import (
	"bytes"
	"context"
	"deadalus-orch/server/internal/infrastructure/db"
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"deadalus-orch/server/internal/pkg/config"
	"deadalus-orch/server/internal/pkg/utils"
	general_command "deadalus-orch/server/internal/usecase/command/general"
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

func (s *RaftStore) Get(ctx context.Context, key string, rate limiter.Rate) (limiter.Context, error) {
	return s.Increment(ctx, key, 1, rate)
}

func (s *RaftStore) Increment(ctx context.Context, key string, quantity int64, rate limiter.Rate) (limiter.Context, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	fullKey := s.fullKey(key)
	now := time.Now().Unix()

	readCmd := general_command.Query_Command{
		Command: general_command.RK_Command{
			Key:                fullKey,
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSelector,
		},
		Now: utils.GetNowInInt(),
	}

	resp, err := s.node.Read(ctx, readCmd)
	if err != nil {
		return limiter.Context{}, err
	}

	var state *models.RateLimitState
	if resp == nil {
		exp := now + int64(s.ttl.Seconds())
		state = &models.RateLimitState{
			Limit:     rate.Limit,
			Remaining: rate.Limit - quantity,
			Reached:   quantity > rate.Limit,
			ExpiredAt: exp,
		}
	} else {
		buf := bytes.NewBuffer(resp.([]byte))
		dec := gob.NewDecoder(buf)
		state = &models.RateLimitState{}
		if err := dec.Decode(state); err != nil {
			return limiter.Context{}, err
		}

		if now >= state.ExpiredAt {
			// La ventana expiró, reseteamos
			state.Limit = rate.Limit
			state.Remaining = rate.Limit - quantity
			state.Reached = quantity > rate.Limit
			state.ExpiredAt = now + int64(s.ttl.Seconds())
		} else {
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
		}
	}

	ttlRemaining := state.ExpiredAt - now
	if ttlRemaining < 1 {
		ttlRemaining = 1
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return limiter.Context{}, err
	}

	writeCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                fullKey,
				Value:              buf.Bytes(),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSelector,
				TTL:                int(ttlRemaining),
				Op:                 general_command.PutOpTTL,
			},
		},
	}

	if _, err := s.node.Write(ctx, writeCmd); err != nil {
		return limiter.Context{}, err
	}

	return limiter.Context{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Reset:     state.ExpiredAt,
		Reached:   state.Reached,
	}, nil
}

func (s *RaftStore) Peek(ctx context.Context, key string, rate limiter.Rate) (limiter.Context, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	fullKey := s.fullKey(key)
	now := time.Now().Unix()

	readCmd := general_command.Query_Command{
		Command: general_command.RK_Command{
			Key:                fullKey,
			ColumnFamilyName:   db.MasterEventFC,
			ColumnFamilySector: db.MasterEventFCSelector,
		},
		Now: utils.GetNowInInt(),
	}

	resp, err := s.node.Read(ctx, readCmd)
	if err != nil {
		return limiter.Context{}, err
	}

	if resp == nil {
		exp := now + int64(s.ttl.Seconds())
		return limiter.Context{
			Limit:     rate.Limit,
			Remaining: rate.Limit,
			Reset:     exp,
			Reached:   false,
		}, nil
	}

	buf := bytes.NewBuffer(resp.([]byte))
	dec := gob.NewDecoder(buf)
	state := &models.RateLimitState{}
	if err := dec.Decode(state); err != nil {
		return limiter.Context{}, err
	}

	if now >= state.ExpiredAt {
		return limiter.Context{
			Limit:     rate.Limit,
			Remaining: rate.Limit,
			Reset:     now + int64(s.ttl.Seconds()),
			Reached:   false,
		}, nil
	}

	return limiter.Context{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Reset:     state.ExpiredAt,
		Reached:   state.Remaining <= 0,
	}, nil
}

func (s *RaftStore) Set(ctx context.Context, key string, c limiter.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	fullKey := s.fullKey(key)
	now := time.Now().Unix()

	state := &models.RateLimitState{
		Limit:     c.Limit,
		Remaining: c.Remaining,
		Reached:   c.Reached,
		ExpiredAt: now + int64(s.ttl.Seconds()),
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return err
	}

	writeCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                fullKey,
				Value:              buf.Bytes(),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSelector,
				TTL:                int(s.ttl.Seconds()),
				Op:                 general_command.PutOpTTL,
			},
		},
	}

	_, err := s.node.Write(ctx, writeCmd)
	return err
}

func (s *RaftStore) Reset(ctx context.Context, key string, rate limiter.Rate) (limiter.Context, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.GlobalConfiguration.ApiRaftTimeout)
	defer cancel()
	fullKey := s.fullKey(key)
	now := time.Now().Unix()

	state := &models.RateLimitState{
		Limit:     rate.Limit,
		Remaining: rate.Limit,
		Reached:   false,
		ExpiredAt: now + int64(s.ttl.Seconds()),
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(state); err != nil {
		return limiter.Context{}, err
	}

	writeCmd := general_command.FSM_Command{
		Now:  utils.GetNowInInt(),
		Type: general_command.RW,
		CMD: general_command.RWK_Command{
			Op: general_command.Write,
			CMD: general_command.WK_Command{
				Key:                fullKey,
				Value:              buf.Bytes(),
				ColumnFamilyName:   db.MasterEventFC,
				ColumnFamilySector: db.MasterEventFCSelector,
				TTL:                int(s.ttl.Seconds()),
				Op:                 general_command.PutOpTTL,
			},
		},
	}

	if _, err := s.node.Write(ctx, writeCmd); err != nil {
		return limiter.Context{}, err
	}

	return limiter.Context{
		Limit:     state.Limit,
		Remaining: state.Remaining,
		Reset:     state.ExpiredAt,
		Reached:   state.Reached,
	}, nil
}
