package binding_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deadalus-orch/server/internal/infrastructure/db"
	bindingCommand "deadalus-orch/server/internal/usecase/command/binding"
	"deadalus-orch/shared/models"
)

const (
	AssertBindingDefaultFC  = "default"
	AssertBindingTestFC     = "test_fc"
	AssertBindingTemporalFC = "temporal_fc"
	AssertBindingTestCFS    = "test-sector"
)

// Helper function to create test Pebble store
func newTestPebbleStoreForAssertBinding(t *testing.T, cfNames []string, ttlCfNames []string) db.KVStore {
	tempDir, err := os.MkdirTemp("", "assert_binding_pebble_test_*")
	require.NoError(t, err)
	t.Logf("Creating Pebble store in: %s", tempDir)

	store, err := db.CreatePebbleStore(tempDir, cfNames, ttlCfNames)
	require.NoError(t, err)
	require.NotNil(t, store)

	t.Cleanup(func() {
		t.Logf("Closing and removing Pebble store from: %s", tempDir)
		store.Close()
		os.RemoveAll(tempDir)
	})
	return store
}

// Helper function to setup test exchange and queue
func setupTestExchangeAndQueue(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) (*models.Exchange, *models.Queue) {
	uow := db.NewUnitOfWork(store, nil)
	idFactory := &db.DeterministicIDGeneratorFactory{}
	// Create VNamespace first
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	vNamespace := &models.VNamespace{
		ID:   "test-namespace-id",
		Name: "test-namespace",
	}
	_, err = vNamespaceRepo.CreateVNamespace(vNamespace, now)
	require.NoError(t, err)

	// Create Exchange
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	exchange := &models.Exchange{
		ID:         "test-exchange-id",
		Code:       "TEST_EXCHANGE",
		Name:       "Test Exchange",
		VNamespace: "test-namespace",
		Type:       models.Direct,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	exchangeID, err := exchangeRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	require.NotEmpty(t, exchangeID)

	// Create Queue
	queueRepo, err := db.NewQueueRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	queue := &models.Queue{
		ID:                        "test-queue-id",
		Code:                      "TEST_QUEUE",
		Name:                      "Test Queue",
		VNamespace:                "test-namespace",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		TTLQueue:                  3600,
		AllowDuplicated:           true,
		MaxAttempts:               3,
		MessagesCount:             0,
		DesiredPriorityThresholds: map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		PriorityThresholds:        map[int]int{1: 100, 2: 50, 3: 30, 10: 10},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	queueID, err := queueRepo.CreateQueue(queue, now)
	require.NoError(t, err)
	require.NotEmpty(t, queueID)

	err = uow.Commit()
	require.NoError(t, err)

	// After commit, reload the entities to get the actual created entities
	uow2 := db.NewUnitOfWork(store, nil)
	exchangeRepo2, err := db.NewExchangeRepository(uow2, idFactory, cf, cfs)
	require.NoError(t, err)

	exchange, err = exchangeRepo2.GetExchangeById(exchangeID, now)
	require.NoError(t, err)
	require.NotNil(t, exchange)

	queueRepo2, err := db.NewQueueRepository(uow2, idFactory, cf, cfs)
	require.NoError(t, err)

	queue, err = queueRepo2.GetQueueById(queueID, now)
	require.NoError(t, err)
	require.NotNil(t, queue)

	return exchange, queue
}

func TestAssertBindingCommand_CreateNewBinding(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Setup exchange and queue
	exchange, queue := setupTestExchangeAndQueue(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

	// Execute command for Direct exchange
	uow := db.NewUnitOfWork(store, nil)

	cmd := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-001",
		Code:         "TEST_BINDING_001",
		QueueCode:    queue.Code,
		ExchangeCode: exchange.Code,
		VNamespace:   "test-namespace",
		RoutingKey:   "test.routing.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result := cmd.Execute(uow, now)
	require.Empty(t, result.Error)
	err := uow.Commit()
	require.NoError(t, err)

	// Verify results
	binding := result.Result.(models.Binding)
	assert.Equal(t, "TEST_BINDING_001", binding.Code)
	assert.Equal(t, exchange.ID, binding.ExchangeID)
	assert.Equal(t, queue.ID, binding.QueueID)
	assert.Equal(t, "test.routing.key", binding.RoutingKey)
	assert.Equal(t, models.BindingTypeClassic, binding.BindingType)
	assert.Equal(t, "test-namespace", binding.VNamespace)
	assert.NotEmpty(t, binding.ID)
}

func TestAssertBindingCommand_UpdateExistingBinding(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Setup exchange and queue
	exchange, queue := setupTestExchangeAndQueue(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

	// Create initial binding
	uow1 := db.NewUnitOfWork(store, nil)

	cmd1 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-002",
		Code:         "TEST_BINDING_002",
		QueueCode:    queue.Code,
		ExchangeCode: exchange.Code,
		VNamespace:   "test-namespace",
		RoutingKey:   "old.routing.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Update binding with new routing key
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-002",
		Code:         "TEST_BINDING_002", // Same code to update existing binding
		ExchangeCode: exchange.Code,
		VNamespace:   "test-namespace",
		RoutingKey:   "new.routing.key",
		BindingType:  models.BindingTypeDynamic, // Dynamic binding doesn't need QueueCode
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	// Verify update
	binding := result2.Result.(models.Binding)
	assert.Equal(t, "new.routing.key", binding.RoutingKey)
	assert.Equal(t, models.BindingTypeDynamic, binding.BindingType)
}

func TestAssertBindingCommand_ValidationErrors(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Test missing ExchangeCode
	uow1 := db.NewUnitOfWork(store, nil)

	cmd1 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-003",
		Code:         "TEST_BINDING_003",
		QueueCode:    "some-queue-code",
		VNamespace:   "test-namespace",
		RoutingKey:   "test.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	assert.NotEmpty(t, result1.Error)
	assert.Contains(t, result1.Error, "ExchangeCode is required")

	// Test missing QueueCode
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-004",
		Code:         "TEST_BINDING_004",
		ExchangeCode: "some-exchange-code",
		VNamespace:   "test-namespace",
		RoutingKey:   "test.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result2 := cmd2.Execute(uow2, now)
	assert.NotEmpty(t, result2.Error)
	assert.Contains(t, result2.Error, "QueueCode is required")

	// Test non-existent exchange
	uow3 := db.NewUnitOfWork(store, nil)
	cmd3 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-005",
		Code:         "TEST_BINDING_005",
		ExchangeCode: "non-existent-exchange",
		QueueCode:    "some-queue-code",
		VNamespace:   "test-namespace",
		RoutingKey:   "test.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result3 := cmd3.Execute(uow3, now)
	assert.NotEmpty(t, result3.Error)
	assert.Contains(t, result3.Error, "Exchange with Code 'non-existent-exchange' in VNamespace 'test-namespace' does not exist")
}

func TestAssertBindingCommand_ExchangeTypeValidation(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Setup different exchange types
	uow := db.NewUnitOfWork(store, nil)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	// Create VNamespace
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, AssertBindingTestFC, AssertBindingTestCFS)
	require.NoError(t, err)
	vNamespace := &models.VNamespace{
		ID:   "test-namespace-id",
		Name: "test-namespace",
	}
	_, err = vNamespaceRepo.CreateVNamespace(vNamespace, now)
	require.NoError(t, err)

	// Create Topic Exchange
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, AssertBindingTestFC, AssertBindingTestCFS)
	require.NoError(t, err)
	topicExchange := &models.Exchange{
		ID:         "test-topic-exchange-id",
		Code:       "TEST_TOPIC_EXCHANGE",
		Name:       "Test Topic Exchange",
		VNamespace: "test-namespace",
		Type:       models.Topic,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err = exchangeRepo.CreateExchange(topicExchange, now)
	require.NoError(t, err)

	// Create Queue
	queueRepo, err := db.NewQueueRepository(uow, idFactory, AssertBindingTestFC, AssertBindingTestCFS)
	require.NoError(t, err)
	queue := &models.Queue{
		ID:                        "test-queue-id",
		Name:                      "Test Queue",
		Code:                      "TEST_QUEUE",
		VNamespace:                "test-namespace",
		State:                     models.QueueActive,
		Type:                      models.StandardQueue,
		TTLQueue:                  0,
		AllowDuplicated:           false,
		MaxAttempts:               3,
		DesiredPriorityThresholds: map[int]int{1: 100, 2: 200},
		PriorityThresholds:        map[int]int{1: 100, 2: 200},
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	_, err = queueRepo.CreateQueue(queue, now)
	require.NoError(t, err)
	err = uow.Commit()
	require.NoError(t, err)

	// Test Topic exchange requires Pattern (should fail without pattern)
	uow1 := db.NewUnitOfWork(store, nil)
	cmd1 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "NewBindingID",
		Code:         "TEST_BINDING_006",
		ExchangeCode: "TEST_TOPIC_EXCHANGE",
		QueueCode:    "TEST_QUEUE",
		VNamespace:   "test-namespace",
		RoutingKey:   "test.key", // Should not be allowed for Topic
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	assert.NotEmpty(t, result1.Error)
	assert.Contains(t, result1.Error, "Pattern is required for Topic exchanges")

	// Test Topic exchange with Pattern (should succeed)
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-007",
		Code:         "TEST_BINDING_007",
		ExchangeCode: "TEST_TOPIC_EXCHANGE",
		QueueCode:    "TEST_QUEUE",
		VNamespace:   "test-namespace",
		Pattern:      "user.*.created",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result2 := cmd2.Execute(uow2, now)
	require.Empty(t, result2.Error)
	err = uow2.Commit()
	require.NoError(t, err)

	binding := result2.Result.(models.Binding)
	assert.Equal(t, "user.*.created", binding.Pattern)
}

func TestAssertBindingCommand_PreventDuplicateClassicBindings(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Setup exchange and queue
	exchange, queue := setupTestExchangeAndQueue(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

	// Create first classic binding
	uow1 := db.NewUnitOfWork(store, nil)

	cmd1 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-008",
		Code:         "TEST_BINDING_008",
		QueueCode:    queue.Code,
		ExchangeCode: exchange.Code,
		VNamespace:   "test-namespace",
		RoutingKey:   "test.routing.key",
		BindingType:  models.BindingTypeClassic,
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result1 := cmd1.Execute(uow1, now)
	require.Empty(t, result1.Error)
	err := uow1.Commit()
	require.NoError(t, err)

	// Try to create second classic binding with same exchange and queue but different code
	uow2 := db.NewUnitOfWork(store, nil)
	cmd2 := &bindingCommand.AssertBindingCommand{
		NewBindingID: "test-binding-id-009",
		Code:         "TEST_BINDING_009", // Different code
		QueueCode:    queue.Code,         // Same queue
		ExchangeCode: exchange.Code,      // Same exchange
		VNamespace:   "test-namespace",
		RoutingKey:   "different.routing.key",
		BindingType:  models.BindingTypeClassic, // Same classic type
		CF:           AssertBindingTestFC,
		CFS:          AssertBindingTestCFS,
	}
	result2 := cmd2.Execute(uow2, now.Add(time.Second))

	// Should fail because classic binding already exists between this exchange and queue
	assert.NotEmpty(t, result2.Error)
	assert.Contains(t, result2.Error, "A classic binding between exchange")
	assert.Contains(t, result2.Error, "already exists with Code 'TEST_BINDING_008'")
}

func TestAssertBindingCommand_RejectDynamicBindingsForFanoutAndTopicExchanges(t *testing.T) {
	store := newTestPebbleStoreForAssertBinding(t, []string{AssertBindingDefaultFC, AssertBindingTestFC, "admin"}, []string{AssertBindingTemporalFC})
	now := time.Now()

	// Test with Fanout exchange
	t.Run("Fanout exchange should be rejected for dynamic binding", func(t *testing.T) {
		// Setup fanout exchange
		fanoutExchange := setupTestFanoutExchange(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

		uow := db.NewUnitOfWork(store, nil)
		cmd := &bindingCommand.AssertBindingCommand{
			NewBindingID:       "test-fanout-binding-id",
			Code:               "TEST_FANOUT_DYNAMIC_BINDING",
			ExchangeCode:       fanoutExchange.Code,
			VNamespace:         "test-namespace-fanout",
			BindingType:        models.BindingTypeDynamic,
			TargetExchangeType: models.TargetExchangeTypeQueue,
			CF:                 AssertBindingTestFC,
			CFS:                AssertBindingTestCFS,
		}

		result := cmd.Execute(uow, now)
		assert.NotEmpty(t, result.Error)
		assert.Contains(t, result.Error, "Dynamic bindings cannot use fanout exchanges")
	})

	// Test with Topic exchange
	t.Run("Topic exchange should be rejected for dynamic binding", func(t *testing.T) {
		// Setup topic exchange
		topicExchange := setupTestTopicExchange(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

		uow := db.NewUnitOfWork(store, nil)
		cmd := &bindingCommand.AssertBindingCommand{
			NewBindingID:       "test-topic-binding-id",
			Code:               "TEST_TOPIC_DYNAMIC_BINDING",
			ExchangeCode:       topicExchange.Code,
			VNamespace:         "test-namespace-topic",
			BindingType:        models.BindingTypeDynamic,
			TargetExchangeType: models.TargetExchangeTypeQueue,
			CF:                 AssertBindingTestFC,
			CFS:                AssertBindingTestCFS,
		}

		result := cmd.Execute(uow, now)
		assert.NotEmpty(t, result.Error)
		assert.Contains(t, result.Error, "Dynamic bindings cannot use topic exchanges")
	})

	// Test with Direct exchange (should succeed)
	t.Run("Direct exchange should be accepted for dynamic binding", func(t *testing.T) {
		// Setup direct exchange
		directExchange := setupTestDirectExchange(t, store, AssertBindingTestFC, AssertBindingTestCFS, now)

		uow := db.NewUnitOfWork(store, nil)
		cmd := &bindingCommand.AssertBindingCommand{
			NewBindingID:       "test-direct-binding-id",
			Code:               "TEST_DIRECT_DYNAMIC_BINDING",
			ExchangeCode:       directExchange.Code,
			VNamespace:         "test-namespace-direct",
			BindingType:        models.BindingTypeDynamic,
			TargetExchangeType: models.TargetExchangeTypeQueue,
			CF:                 AssertBindingTestFC,
			CFS:                AssertBindingTestCFS,
		}

		result := cmd.Execute(uow, now)
		require.Empty(t, result.Error, "Expected no error but got: %s", result.Error)
		err := uow.Commit()
		require.NoError(t, err)
	})
}

// Helper function to setup test fanout exchange
func setupTestFanoutExchange(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Exchange {
	uow := db.NewUnitOfWork(store, nil)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	// Create VNamespace first
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	vNamespace := &models.VNamespace{
		ID:   "test-namespace-fanout-id",
		Name: "test-namespace-fanout",
	}
	_, err = vNamespaceRepo.CreateVNamespace(vNamespace, now)
	require.NoError(t, err)

	// Create exchange repository
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	// Create fanout exchange
	exchange := &models.Exchange{
		ID:         "test-fanout-exchange-id",
		Code:       "TEST_FANOUT_EXCHANGE",
		Name:       "Test Fanout Exchange",
		Type:       models.Fanout,
		VNamespace: "test-namespace-fanout",
	}

	_, err = exchangeRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	err = uow.Commit()
	require.NoError(t, err)

	return exchange
}

// Helper function to setup test topic exchange
func setupTestTopicExchange(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Exchange {
	uow := db.NewUnitOfWork(store, nil)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	// Create VNamespace first
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	vNamespace := &models.VNamespace{
		ID:   "test-namespace-topic-id",
		Name: "test-namespace-topic",
	}
	_, err = vNamespaceRepo.CreateVNamespace(vNamespace, now)
	require.NoError(t, err)

	// Create exchange repository
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	// Create topic exchange
	exchange := &models.Exchange{
		ID:         "test-topic-exchange-id",
		Code:       "TEST_TOPIC_EXCHANGE",
		Name:       "Test Topic Exchange",
		Type:       models.Topic,
		VNamespace: "test-namespace-topic",
	}

	_, err = exchangeRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	err = uow.Commit()
	require.NoError(t, err)

	return exchange
}

// Helper function to setup test direct exchange
func setupTestDirectExchange(t *testing.T, store db.KVStore, cf, cfs string, now time.Time) *models.Exchange {
	uow := db.NewUnitOfWork(store, nil)

	idFactory := &db.DeterministicIDGeneratorFactory{}
	// Create VNamespace first
	vNamespaceRepo, err := db.NewVNamespaceRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	vNamespace := &models.VNamespace{
		ID:   "test-namespace-direct-id",
		Name: "test-namespace-direct",
	}
	_, err = vNamespaceRepo.CreateVNamespace(vNamespace, now)
	require.NoError(t, err)

	// Create exchange repository
	exchangeRepo, err := db.NewExchangeRepository(uow, idFactory, cf, cfs)
	require.NoError(t, err)

	// Create direct exchange
	exchange := &models.Exchange{
		ID:         "test-direct-exchange-id",
		Code:       "TEST_DIRECT_EXCHANGE",
		Name:       "Test Direct Exchange",
		Type:       models.Direct,
		VNamespace: "test-namespace-direct",
	}

	_, err = exchangeRepo.CreateExchange(exchange, now)
	require.NoError(t, err)
	err = uow.Commit()
	require.NoError(t, err)

	return exchange
}
