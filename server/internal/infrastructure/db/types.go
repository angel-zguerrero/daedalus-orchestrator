package db

// KeyValuePair represents a single key-value pair.
// This struct is often used for returning results from database queries or iterations.
type KeyValuePair struct {
	Key   string // The key as a string.
	Value []byte // The value as a byte slice.
}

// PathProvider is an interface for determining the database storage path.
type PathProvider interface {
	// GetDatabasePath returns the path where the database should be stored.
	// It returns an error if the path cannot be determined or accessed.
	GetDatabasePath() (string, error)
}

// DefaultPathProvider is the default implementation of PathProvider.
// It determines the database path based on the environment:
// - In development (ENV is "development" or not set), it uses a subdirectory in the user's home directory (`~/.daedalus/data`).
// - In other environments (e.g., production), it uses `/var/lib/daedalus/data` and attempts to create it if it doesn't exist.
type DefaultPathProvider struct{}
