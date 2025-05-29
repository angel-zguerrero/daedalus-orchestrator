package dragonboat

import "deadalus-orch/server/internal/infrastructure/db"

// NodeRole is a string type representing the role of a node in the cluster.
// Examples of roles include "consensus", "scheduler", "connector".
type NodeRole string

// Member defines the network identity of a node in the Dragonboat cluster.
// It consists of an IP address and a port number.
type Member struct {
	IP   string // The IP address of the member node.
	Port int    // The port number on which the member node listens for Dragonboat communication.
}

// PagedResultKV represents a paginated result set for key-value queries.
// It contains a slice of key-value pairs and a cursor to fetch the next page of results.
type PagedResultKV struct {
	Data       []db.KeyValuePair // Data holds a slice of KeyValuePair structs for the current page.
	NextCursor []byte            // NextCursor is an opaque byte slice that can be used to fetch the subsequent page of results.
	                           // If NextCursor is empty or nil, it indicates that there are no more pages.
}
