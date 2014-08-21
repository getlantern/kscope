package kscope

import (
	"strings"
)

const (
	ADDRESS_DELIM      = "|"
	NODE_CHANNEL_DEPTH = 100
)

var (
	DEFAULT_REACHES = []float64{1, .25, .1, .05}
)

// Identifies a node (must not contain ADDRESS_DELIM)
type nodeid string

// source is the combination of an origin nodeid (the origin of the ad) and a
// forwarder nodeid (the most proximate node to forward an ad)
type source string

func sourceFor(origin nodeid, forwarder nodeid) source {
	return source(origin + ADDRESS_DELIM + forwarder)
}

func (s source) origin() nodeid {
	return nodeid(strings.Split(string(s), ADDRESS_DELIM)[0])
}

func (s source) forwarder() nodeid {
	return nodeid(strings.Split(string(s), ADDRESS_DELIM)[1])
}

type ad struct {
	src     source
	degree  int
	payload interface{}
}
