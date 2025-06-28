package models

type RateLimitState struct {
	Limit     int64
	Remaining int64
	Reset     int64
	Reached   bool
}
