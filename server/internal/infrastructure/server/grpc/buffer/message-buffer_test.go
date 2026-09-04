package buffer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type dummyItem struct {
	id string
}

func (d dummyItem) GetGroupKey() string {
	return "dummy-group"
}

func TestMessageBuffer_FlushesWithBackgroundContextWhenItemContextCanceled(t *testing.T) {
	var receivedCtxs []context.Context
	var receivedItems [][]dummyItem
	var mu sync.Mutex

	flushedChan := make(chan struct{}, 10)

	flushFunc := func(ctx context.Context, items []dummyItem) {
		mu.Lock()
		receivedCtxs = append(receivedCtxs, ctx)
		receivedItems = append(receivedItems, items)
		mu.Unlock()
		flushedChan <- struct{}{}
	}

	logger := zerolog.Nop()
	mb := NewMessageBuffer[dummyItem](100*time.Millisecond, 10, logger, flushFunc)
	defer mb.Stop()

	// Create a context that is immediately canceled
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Add item using canceled context
	item := dummyItem{id: "item-1"}
	mb.Add(canceledCtx, item)

	select {
	case <-flushedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for buffer flush")
	}

	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, receivedCtxs, 1)
	assert.Nil(t, receivedCtxs[0].Err(), "flushFunc context should not be canceled")
	assert.Len(t, receivedItems, 1)
	assert.Equal(t, "item-1", receivedItems[0][0].id)
}

func TestMessageBuffer_ConcurrentAddWithCanceledContexts(t *testing.T) {
	var totalFlushedCount int
	var flushedCtxErrors []error
	var mu sync.Mutex

	flushedChan := make(chan struct{}, 100)

	flushFunc := func(ctx context.Context, items []dummyItem) {
		mu.Lock()
		totalFlushedCount += len(items)
		flushedCtxErrors = append(flushedCtxErrors, ctx.Err())
		mu.Unlock()
		flushedChan <- struct{}{}
	}

	logger := zerolog.Nop()
	mb := NewMessageBuffer[dummyItem](10*time.Millisecond, 5, logger, flushFunc)
	defer mb.Stop()

	const numWorkers = 10
	const itemsPerWorker = 20
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < itemsPerWorker; j++ {
				canceledCtx, cancel := context.WithCancel(context.Background())
				cancel() // cancel context before calling Add
				mb.Add(canceledCtx, dummyItem{id: "item"})
			}
		}(i)
	}

	wg.Wait()

	// Wait for all items to be processed by flusher
	timeout := time.After(3 * time.Second)
	for {
		mu.Lock()
		count := totalFlushedCount
		mu.Unlock()

		if count >= numWorkers*itemsPerWorker {
			break
		}

		select {
		case <-flushedChan:
		case <-timeout:
			t.Fatalf("timed out waiting for all items to flush. Got %d/%d", count, numWorkers*itemsPerWorker)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, numWorkers*itemsPerWorker, totalFlushedCount)
	for _, err := range flushedCtxErrors {
		assert.NoError(t, err, "flushFunc context should never be canceled")
	}
}
