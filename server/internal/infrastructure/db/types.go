package db

// KeyValuePair represents a single key-value pair.
// This struct is often used for returning results from database queries or iterations.
type KeyValuePair struct {
	Key   string // The key as a string.
	Value []byte // The value as a byte slice.
}
