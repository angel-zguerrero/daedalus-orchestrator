package dragonboat

type NodeRole string

type Member struct {
	IP   string
	Port int
}
type PagedResult struct {
	Data       [][]byte
	NextCursor []byte
}
