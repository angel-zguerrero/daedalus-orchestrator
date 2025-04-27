package dragonboat_test

import (
	"deadalus-orch/server/internal/infrastructure/dragonboat"
	"testing"
	"time"
)

func TestOne(t *testing.T) {

	dragonboat.Init(1, 1, "3435")
	//-----

	dragonboat.Init(1, 2, "3436")

	time.Sleep(240 * time.Second)
}
