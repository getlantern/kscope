package kscope

import (
	"math"
	"math/rand"
	"time"
)

var (
	RAND = rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
)

// node is a node in the trust graph
type node struct {
	kscope        *KScope
	id            nodeid
	trusted       []nodeid
	trustedCh     chan nodeid
	untrustedCh   chan nodeid
	resetRoutesCh chan interface{}
	adCh          chan *ad
	routes        map[nodeid][]nodeid
}

func newNode(kscope *KScope, id nodeid) *node {
	return &node{
		kscope:        kscope,
		id:            id,
		trusted:       make([]nodeid, 0),
		trustedCh:     make(chan nodeid, NODE_CHANNEL_DEPTH),
		untrustedCh:   make(chan nodeid, NODE_CHANNEL_DEPTH),
		resetRoutesCh: make(chan interface{}, NODE_CHANNEL_DEPTH),
		adCh:          make(chan *ad, NODE_CHANNEL_DEPTH),
		routes:        make(map[nodeid][]nodeid),
	}
}

func (node *node) trust(id nodeid) {
	node.trustedCh <- id
}

func (node *node) untrust(id nodeid) {
	node.untrustedCh <- id
}

func (node *node) resetRoutes() {
	node.resetRoutesCh <- nil
}

func (node *node) ad(ad *ad) {
	node.adCh <- ad
}

func (node *node) process() {
	for {
		select {
		case id := <-node.trustedCh:
			node.processTrusted(id)
		case id := <-node.untrustedCh:
			node.processUntrusted(id)
		case <-node.resetRoutesCh:
			node.processResetRoutes()
		case ad := <-node.adCh:
			node.processAd(ad)
		}
	}
}

func (node *node) processTrusted(id nodeid) {
	node.trusted = append(node.trusted, id)
}

func (node *node) processUntrusted(id nodeid) {
	node.trusted = removeFrom(node.trusted, id)
	for src, dests := range node.routes {
		if src == id {
			// Remove all routes from the untrusted node
			delete(node.routes, id)
		} else {
			// Remove any routes to the untrusted node
			node.routes[id] = removeFrom(dests, id)
		}
	}
}

func (node *node) processResetRoutes() {
	node.routes = make(map[nodeid][]nodeid)
}

func (node *node) processAd(ad *ad) {
	if !node.trusts(ad.forwarder) {
		return
	}
	node.deliverAd(ad)
	node.forwardAd(ad)
}

func (node *node) deliverAd(ad *ad) {
	if node.id != ad.forwarder && node.id != ad.origin {
		node.kscope.deliverAd(node.id, ad)
	}
}

func (node *node) forwardAd(a *ad) {
	if a.degree >= len(node.kscope.Spreads) {
		// Maximum degree reached, do not forward
		return
	}

	destinations := node.destinationsFor(a)

	forwardedAd := &ad{
		origin:    a.origin,
		forwarder: node.id,
		degree:    a.degree + 1,
		payload:   a.payload,
	}

	for _, destination := range destinations {
		node.kscope.nodeFor(destination).ad(forwardedAd)
	}
}

func (node *node) destinationsFor(ad *ad) []nodeid {
	destinations := node.routes[ad.forwarder]

	// Don't even send ad to forwarder
	nodeids := removeFrom(node.trusted, ad.forwarder)

	// Figure out how many destinations to forward to based on spread
	spread := node.kscope.Spreads[ad.degree]
	if spread == 1 {
		// Send to all trusted nodes other than forwarder
		destinations = nodeids
	} else {
		if destinations == nil {
			destinations = make([]nodeid, 0)
		}

		n := int(math.Ceil(spread * float64(len(nodeids))))

		// Add random destinations until we've reached desired size
		dn := n - len(destinations)

		for i := 0; i < dn; i++ {
			r := len(nodeids) - 1
			if r == 0 {
				destinations = append(destinations, nodeids[0])
				break
			}
			randomDestination := nodeids[RAND.Intn(r)]
			nodeids = removeFrom(nodeids, randomDestination)
			destinations = append(destinations, randomDestination)
		}
	}

	// Remember destinations for later
	node.routes[ad.forwarder] = destinations
	return destinations
}

func (node *node) trusts(id nodeid) bool {
	if id == node.id {
		// We always trust ourselves
		return true
	}
	for _, trustedId := range node.trusted {
		if id == trustedId {
			return true
		}
	}
	return false
}

func removeFrom(nodeids []nodeid, id nodeid) []nodeid {
	for i, existingId := range nodeids {
		if id == existingId {
			result := make([]nodeid, len(nodeids)-1)
			copy(result, nodeids[:i])
			copy(result[i:], nodeids[i+1:])
			return result
		}
	}
	return nodeids
}
