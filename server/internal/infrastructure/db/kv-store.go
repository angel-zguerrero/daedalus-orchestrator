package db

// WriteBatch represents a batch of write operations to be applied atomically.
// The concrete implementation of this interface will depend on the underlying database system.
type WriteBatch interface {
	// AddPut adds a Put operation to the batch.
	// Put(columnFamily string, key string, value []byte)
	// AddDelete adds a Delete operation to the batch.
	// Delete(columnFamily string, key string)
}

// KVStore is an interface for a key-value store.
// It provides basic CRUD operations (Create, Read, Update, Delete) as well as
// other utility functions for interacting with the underlying data store.
type KVStore interface {
	// Get retrieves the value associated with a key from a specific column family.
	// Parameters:
	//   - columnFamily: The name of the column family.
	//   - key: The key to retrieve.
	// Returns:
	//   - The value as a byte slice, or nil if the key is not found.
	//   - An error if any occurred during the operation.
	Get(columnFamily string, key string) ([]byte, error)

	// Put stores a key-value pair into a specific column family.
	// If the key already exists, its value will be overwritten.
	// Parameters:
	//   - columnFamily: The name of the column family.
	//   - key: The key to store.
	//   - value: The value to store as a byte slice.
	// Returns:
	//   - An error if any occurred during the operation.
	Put(columnFamily string, key string, value []byte) error

	// Write applies a batch of operations (e.g., Puts, Deletes) atomically.
	// The exact type of `batch` depends on the KVStore implementation.
	// Parameters:
	//   - batch: A WriteBatch object containing the operations to apply.
	// Returns:
	//   - An error if any occurred during the operation.
	Write(batch interface{}) error // TODO: Use a more specific type like WriteBatch if possible

	// DumpAll retrieves all key-value pairs from the store.
	// The format of the returned data (interface{}) is implementation-specific.
	// This method is typically used for debugging or backup purposes and might be memory-intensive
	// for large datasets.
	// Returns:
	//   - An interface{} containing all data, or nil if the store is empty.
	//   - An error if any occurred during the operation.
	DumpAll() (interface{}, error)

	// Iterate executes a function for each key-value pair in the store across all column families.
	// The iteration order is not guaranteed unless specified by the implementation.
	// Parameters:
	//   - fn: A function to be called for each key-value pair.
	//         It receives the column family name, key, and value.
	//         If this function returns an error, the iteration stops and the error is returned.
	// Returns:
	//   - An error if any occurred during iteration or if `fn` returned an error.
	Iterate(fn func(cfName string, key, value []byte) error) error

	// ClearAll removes all data from the key-value store. This is a destructive operation.
	// Returns:
	//   - An error if any occurred during the operation.
	ClearAll() error

	// Flush forces any buffered data to be written to the underlying storage.
	// Returns:
	//   - An error if any occurred during the operation.
	Flush() error

	// Close closes the connection to the key-value store, releasing any resources.
	// After calling Close, other methods should not be called.
	// Returns:
	//   - An error if any occurred during the closing process.
	Close() error
}
