package db

// Internal key prefixes for RocksDB data organization.
const (
	// PrefixTTLIndex is used for keys that are part of a TTL index (e.g., mapping a primary key to its TTL expiration time).
	PrefixTTLIndex = "__meta:ttl:ttl-index:"
	// PrefixTTLExpire is used for keys that track TTL expiration times (e.g., mapping an expiration time to a set of primary keys).
	PrefixTTLExpire = "__meta:ttl:ttl-expire:"
)

const (
	DefaultFC = "default"
	MetaFC    = "meta"
)

const (
	// MetaFCSelector is the selector for the metadata column family.
	MetaFCSelector = MetaFC + "-selector"
	// DefaultFCSelector is the selector for the default column family.
	DefaultFCSelector = DefaultFC + "-selector"
)
