package kscope

import (
	"sync"
)

type KScope struct {
	Deliver    func(id string, payload interface{})
	Spreads    []float64
	nodes      map[nodeid]*node
	nodesMutex sync.Mutex
}

func (kscope *KScope) Start() {
	if kscope.Spreads == nil {
		kscope.Spreads = DEFAULT_SPREADS
	}
	kscope.nodes = make(map[nodeid]*node)
}

func (kscope *KScope) Trust(truster string, trustee string) {
	kscope.nodeFor(nodeid(truster)).trust(nodeid(trustee))
}

func (kscope *KScope) Untrust(truster string, trustee string) {
	kscope.nodeFor(nodeid(truster)).untrust(nodeid(trustee))
}

func (kscope *KScope) Advertise(node string, payload interface{}) {
	origin := nodeid(node)
	ad := &ad{
		origin:    origin,
		forwarder: origin,
		degree:    0,
		payload:   payload,
	}
	kscope.nodeFor(origin).ad(ad)
}

func (kscope *KScope) nodeFor(id nodeid) *node {
	kscope.nodesMutex.Lock()
	defer kscope.nodesMutex.Unlock()
	node := kscope.nodes[id]
	if node == nil {
		node = newNode(kscope, id)
		go node.process()
		kscope.nodes[id] = node
	}
	return node
}

func (kscope *KScope) deliverAd(dst nodeid, ad *ad) {
	if kscope.Deliver != nil {
		kscope.Deliver(string(dst), ad.payload)
	}
}
