package client

import (
	"time"

	digest "github.com/opencontainers/go-digest"
)

type Vertex struct {
	Digest    digest.Digest
	Inputs    []digest.Digest
	Name      string
	Started   *time.Time
	Completed *time.Time
	Cached    bool
	Error     string
	Parent    digest.Digest
}

type VertexStatus struct {
	ID        string
	Vertex    digest.Digest
	Name      string
	Total     int64
	Current   int64
	Timestamp time.Time
	Started   *time.Time
	Completed *time.Time
}

type VertexLog struct {
	Vertex    digest.Digest
	Stream    int
	Data      []byte
	Timestamp time.Time
}

type SolveStatus struct {
	Vertexes []*Vertex
	Statuses []*VertexStatus
	Logs     []*VertexLog
}

//
// type VertexEvent struct {
// 	ID        digest.Digest
// 	Vertex    digest.Digest
// 	Name      string
// 	Total     int
// 	Current   int
// 	Timestamp int64
// }
