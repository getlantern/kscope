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
	routes        map[source][]nodeid
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
		routes:        make(map[source][]nodeid),
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
	t := node.trusted
	for i, trustedId := range t {
		if id == trustedId {
			copy(t[i:], t[i+1:])
			t[len(t)-1] = ""
			node.trusted = t[:len(t)-1]
			return
		}
	}
}

func (node *node) processResetRoutes() {
	node.routes = make(map[source][]nodeid)
}

func (node *node) processAd(ad *ad) {
	if !node.trusts(ad.src.forwarder()) {
		return
	}
	node.deliverAd(ad)
	node.forwardAd(ad)
}

func (node *node) deliverAd(ad *ad) {
	if node.id != ad.src.forwarder() && node.id != ad.src.origin() {
		node.kscope.deliverAd(node.id, ad)
	}
}

func (node *node) forwardAd(a *ad) {
	if a.degree >= len(node.kscope.Reaches) {
		// Maximum degree reached, do not forward
		return
	}

	destinations := node.destinationsFor(a)

	forwardedAd := &ad{
		src:     sourceFor(a.src.origin(), node.id),
		degree:  a.degree + 1,
		payload: a.payload,
	}
	for _, destination := range destinations {
		node.kscope.nodeFor(destination).ad(forwardedAd)
	}
}

func (node *node) destinationsFor(ad *ad) []nodeid {
	destinations := node.routes[ad.src]
	if destinations == nil {
		destinations = make([]nodeid, 0)
	}

	// Don't bother forwarding ad to origin or forwarder
	nodeids := idsWithout(node.trusted, ad.src.forwarder())
	nodeids = idsWithout(nodeids, ad.src.origin())

	// Figure out how many destinations to forward to based on reach
	reach := node.kscope.Reaches[ad.degree]
	if reach == 1 {
		destinations = nodeids
	} else {
		n := int(math.Ceil(reach * float64(len(nodeids))))

		// Add random destinations until we've reached desired size
		dn := n - len(destinations)
		for i := 0; i < dn; i++ {
			r := len(nodeids) - 1
			if r == 0 {
				destinations = append(destinations, nodeids[0])
				break
			}
			randomDestination := nodeids[RAND.Intn(r)]
			nodeids = idsWithout(nodeids, randomDestination)
			destinations = append(destinations, randomDestination)
		}
	}

	// Remember destinations for later
	node.routes[ad.src] = destinations
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

func idsWithout(nodeids []nodeid, id nodeid) []nodeid {
	result := make([]nodeid, len(nodeids))
	skipped := 0
	for i, trustedId := range nodeids {
		if id == trustedId {
			skipped = skipped + 1
		} else {
			result[i-skipped] = trustedId
		}

	}
	return result[:len(result)-skipped]
}
