package db

// Internal key prefixes for RocksDB data organization.
const (
	// PrefixData is used for regular key-value data.
	PrefixData = "data:"
	// PrefixTTLIndex is used for keys that are part of a TTL index (e.g., mapping a primary key to its TTL expiration time).
	PrefixTTLIndex = "ttl-index:"
	// PrefixTTLExpire is used for keys that track TTL expiration times (e.g., mapping an expiration time to a set of primary keys).
	PrefixTTLExpire = "ttl-expire:"
)

const (
	DefaultFC = "default"
	MetaFC    = "meta"
)
