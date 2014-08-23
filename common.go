package kscope

const (
	NODE_CHANNEL_DEPTH = 100
)

var (
	DEFAULT_SPREADS = []float64{1, .25, .1, .05}
)

// Identifies a node
type nodeid string

type ad struct {
	origin    nodeid
	forwarder nodeid
	degree    int
	payload   interface{}
}
