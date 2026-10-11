package common

import (
	"context"
)

type CallerIdentity struct {
	UserID         string
	AccountID      string
	AccountName    string
	ExternalUserID string
}

const (
	CallerIdentityKey contextKey = "caller_identity"
)

func GetCallerIdentity(ctx context.Context) *CallerIdentity {
	if ctx == nil {
		return nil
	}
	identity, ok := ctx.Value(CallerIdentityKey).(*CallerIdentity)
	if !ok {
		return nil
	}
	return identity
}

func SetCallerIdentity(ctx context.Context, identity *CallerIdentity) context.Context {
	return context.WithValue(ctx, CallerIdentityKey, identity)
}
