package kscope

import (
	"fmt"
	"testing"
	"time"
)

func TestIt(t *testing.T) {
	delivered := make(map[string]string)

	kscope := &KScope{
		Deliver: func(id string, payload interface{}) {
			delivered[id] = payload.(string)
		},
	}
	kscope.Start()

	root := "root"

	// runTest sends an ad and tests that it was delivered correctly
	runTest := func(message string, numExpected int, l1expected []string) {
		time.Sleep(500 * time.Millisecond)
		kscope.Advertise(root, message)
		time.Sleep(500 * time.Millisecond)

		if len(delivered) != numExpected {
			t.Errorf("%s should have been delivered to %d nodes, was delivered to %d", message, numExpected, len(delivered))
		}

		for _, id := range l1expected {
			if delivered[id] != message {
				t.Errorf("Node %s should have gotten an ad for %s", id, message)
			}
		}
	}

	buildTrustHierarchy(kscope, root, [][]int{
		[]int{0, 6},
		[]int{0, 4},
		[]int{0, 10},
		[]int{0, 20},
		[]int{0, 1},
	})
	// Test initial delivery
	runTest("Initial message", 12, []string{"0", "2", "4"})
	// Same nodes should get 2nd message because routes are static
	runTest("Follow up message to same recipients", 12, []string{"0", "2", "4"})

	// Add some more trust relationships
	addedTrustRanges := [][]int{
		[]int{6, 7},
		[]int{4, 5},
		[]int{10, 11},
		[]int{20, 21},
		[]int{1, 2},
	}
	buildTrustHierarchy(kscope, root, addedTrustRanges)
	runTest("Message after adding some more trust relationships", 16, []string{"0", "2", "4", "6"})

	decreaseTrustHierarchy(kscope, root, addedTrustRanges)
	oldDelivered := delivered
	delivered = make(map[string]string)
	runTest("Message after removing some trust relationships", 12, []string{"0", "2", "4"})

	// Everyone who got this message should have gotten the previous too
	for id, _ := range delivered {
		if oldDelivered[id] == "" {
			t.Errorf("Node %s got this message but not the previous one", id)
		}
	}

	// Reset routes
	for _, node := range kscope.nodes {
		node.resetRoutes()
	}
	oldDelivered = delivered
	delivered = make(map[string]string)
	runTest("Message after resetting routes", 12, []string{"0", "2", "4"})

	// Since routes were reset and rebuilt, we should get some at least partly
	// different recipients
	foundDifferent := false
	for id, _ := range delivered {
		if oldDelivered[id] == "" {
			foundDifferent = true
			break
		}
	}
	if !foundDifferent {
		t.Errorf("Delivery after resetting routes should have resulted in different set of recipients")
	}
}

func buildTrustHierarchy(kscope *KScope, root string, ranges [][]int) {
	for i := ranges[0][0]; i < ranges[0][1]; i++ {
		l1 := fmt.Sprintf("%d", i)
		kscope.Trust(root, l1)
		if i%2 == 0 {
			// Only even nodes trust root
			kscope.Trust(l1, root)
		}
		for j := ranges[1][0]; j < ranges[1][1]; j++ {
			l2 := fmt.Sprintf("%d%d", i, j)
			kscope.Trust(l1, l2)
			kscope.Trust(l2, l1)
			for k := ranges[2][0]; k < ranges[2][1]; k++ {
				l3 := fmt.Sprintf("%d%d%d", i, j, k)
				kscope.Trust(l2, l3)
				kscope.Trust(l3, l2)
				for l := ranges[3][0]; l < ranges[3][1]; l++ {
					l4 := fmt.Sprintf("%d%d%d%d", i, j, k, l)
					kscope.Trust(l3, l4)
					kscope.Trust(l4, l3)
					for m := ranges[4][0]; m < ranges[4][1]; m++ {
						l5 := fmt.Sprintf("%d%d%d%d$d", i, j, k, l, m)
						kscope.Trust(l4, l5)
						kscope.Trust(l5, l4)
					}
				}
			}
		}
	}
}

func decreaseTrustHierarchy(kscope *KScope, root string, ranges [][]int) {
	for i := ranges[0][0]; i < ranges[0][1]; i++ {
		l1 := fmt.Sprintf("%d", i)
		kscope.Untrust(root, l1)
		for j := ranges[1][0]; j < ranges[1][1]; j++ {
			l2 := fmt.Sprintf("%d%d", i, j)
			kscope.Untrust(l1, l2)
			for k := ranges[2][0]; k < ranges[2][1]; k++ {
				l3 := fmt.Sprintf("%d%d%d", i, j, k)
				kscope.Untrust(l2, l3)
				for l := ranges[3][0]; l < ranges[3][1]; l++ {
					l4 := fmt.Sprintf("%d%d%d%d", i, j, k, l)
					kscope.Untrust(l3, l4)
					for m := ranges[4][0]; m < ranges[4][1]; m++ {
						l5 := fmt.Sprintf("%d%d%d%d$d", i, j, k, l, m)
						kscope.Untrust(l4, l5)
					}
				}
			}
		}
	}
}
