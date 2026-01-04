package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/krisalay/distributed-systems-journal/distributedclock/hlc"
	"github.com/krisalay/distributed-systems-journal/distributedclock/kvdemo"
)

type Node struct {
	ID    string
	Clock *hlc.Clock
	Store *kvdemo.Store
	Peers []*Node
}

// Put writes a value locally and replicates asynchronously
func (n *Node) Put(key, value string) {
	ts := n.Clock.Now()
	ts.Uncertainty = n.Clock.Uncertainty()

	// Apply locally
	n.Store.Apply(key, kvdemo.Value{
		Data: value,
		TS:   ts,
	})

	// async replication
	for _, peer := range n.Peers {
		go n.send(peer, key, value, ts)
	}
}

// Receive a replication message from a peer
func (n *Node) Receive(key, value string, ts hlc.Timestamp, rtt int64) {
	n.Clock.Update(ts, rtt)
	n.Store.Apply(key, kvdemo.Value{
		Data: value,
		TS:   ts,
	})
}

// send simulates network send with random RTT
func (n *Node) send(peer *Node, key, value string, ts hlc.Timestamp) {
	rtt := rand.Int63n(50) + 10 // 10–60ms
	time.Sleep(time.Duration(rtt) * time.Millisecond)
	peer.Receive(key, value, ts, rtt)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	nodeA := &Node{
		ID:    "A",
		Clock: hlc.New(hlc.Config{MaxClockDriftMillis: 5}),
		Store: kvdemo.NewStore(),
	}
	nodeB := &Node{
		ID:    "B",
		Clock: hlc.New(hlc.Config{MaxClockDriftMillis: 5}),
		Store: kvdemo.NewStore(),
	}

	nodeA.Peers = []*Node{nodeB}
	nodeB.Peers = []*Node{nodeA}

	examDuration := int64(60 * 60 * 1000) // 60 min
	serverEndTime := nodeA.Clock.Now()
	serverEndTime.Physical += examDuration

	fmt.Println("Exam session started (server authoritative end time):", serverEndTime.Physical)

	rttSim := int64(25)
	candidateLeft := serverEndTime.Physical - (nodeA.Clock.Now().Physical + rttSim/2)
	proctorLeft := serverEndTime.Physical - (nodeB.Clock.Now().Physical + rttSim/2)

	fmt.Printf("Initial Time Left: Candidate=%dms, Proctor=%dms\n", candidateLeft, proctorLeft)

	// concurrent writes
	nodeA.Put("user:1", "Alice")
	nodeB.Put("user:1", "Bob")

	time.Sleep(500 * time.Millisecond)

	fmt.Println("\nFinal state after replication:")
	fmt.Println("Node A sees:", nodeA.Store.Data())
	fmt.Println("Node B sees:", nodeB.Store.Data())

	valA := nodeA.Store.Data()["user:1"]
	valB := nodeB.Store.Data()["user:1"]

	fmt.Printf("\nHLC Timestamps with uncertainty (±ms):\n")
	fmt.Printf(" Node A: %d ±%dms\n", valA.TS.Physical, valA.TS.Uncertainty)
	fmt.Printf(" Node B: %d ±%dms\n", valB.TS.Physical, valB.TS.Uncertainty)
}
