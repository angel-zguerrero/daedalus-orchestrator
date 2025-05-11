package dragonboat

const (
	AppliedIndexKey    string = "disk_kv_applied_index"
	CurrentDBFilename  string = "current"
	UpdatingDBFilename string = "current.updating"
)

const (
	prefixData      = "data:"
	prefixTTLIndex  = "ttl-index:"
	prefixTTLExpire = "ttl-expire:"
)
