package hashring

import (
	"fmt"
	"math"
	"sync"
	"testing"
)

// Balance with 2 equal nodes
func TestBalance(t *testing.T) {
	r := New()
	r.AddNode("n1")
	r.AddNode("n2")

	count := make(map[string]int)
	const N = 1_000_000

	for i := 0; i < N; i++ {
		n := string(r.GetNode(fmt.Sprintf("key-%d", i)))
		count[n]++
	}

	c1 := float64(count["n1"]) / float64(N) * 100
	c2 := float64(count["n2"]) / float64(N) * 100
	t.Logf("Balance n1: %.2f%%, n2: %.2f%% (target 50/50)", c1, c2)

	// Allow realistic variance ±5%
	if math.Abs(c1-50) > 5 || math.Abs(c2-50) > 5 {
		t.Fatalf("unbalanced: got %.2f/%.2f", c1, c2)
	}
}

// Removing a node removes all its virtual points
func TestRemoveBalance(t *testing.T) {
	r := New()
	r.AddNode("n1")
	r.AddNode("n2")

	r.RemoveNode("n1")

	if _, exists := r.nodes["n1"]; exists {
		t.Fatalf("node n1 not removed from nodes map")
	}

	// Ensure no ring point maps to removed node
	for _, p := range r.ring {
		if r.nodeMap[p] == "n1" {
			t.Fatalf("found virtual node of removed node n1")
		}
	}
}

// Weights approximate 1:2 ratio
func TestWeighted(t *testing.T) {
	r := New()
	r.AddNodeWeighted("n1", 1)
	r.AddNodeWeighted("n2", 2)

	count := make(map[string]int)
	const N = 1_000_000

	for i := 0; i < N; i++ {
		n := string(r.GetNode(fmt.Sprintf("key-%d", i)))
		count[n]++
	}

	p1 := float64(count["n1"]) / float64(N) * 100
	p2 := float64(count["n2"]) / float64(N) * 100
	t.Logf("Weighted n1: %.2f%%, n2: %.2f%% (target ~33.3/66.7)", p1, p2)

	if math.Abs(p1-33.3) > 5 || math.Abs(p2-66.7) > 5 {
		t.Fatalf("bad weight distribution: %.2f/%.2f", p1, p2)
	}
}

// Replication returns distinct nodes (bounded by cluster size)
func TestReplicas(t *testing.T) {
	r := New()
	r.AddNode("n1")
	r.AddNode("n2")
	r.AddNode("n3")

	nodes := r.GetNodes("key", 3)

	if len(nodes) != 3 {
		t.Fatalf("wrong len: %d", len(nodes))
	}
	if unique(nodes) != 3 {
		t.Fatalf("replicas not unique: %v", nodes)
	}

	t.Logf("Replicas: %v", nodes)
}

// Replication caps at available nodes
func TestReplicaCap(t *testing.T) {
	r := New()
	r.AddNode("n1")
	r.AddNode("n2")

	nodes := r.GetNodes("key", 5)
	if len(nodes) != 2 {
		t.Fatalf("expected cap at 2, got %d", len(nodes))
	}
}

// Race-safety & distribution under concurrent access
func TestConcurrent(t *testing.T) {
	r := New()
	r.AddNode("n1")
	r.AddNode("n2")
	r.AddNode("n3")

	const goroutines = 50
	const keysPerGoroutine = 20_000

	var wg sync.WaitGroup
	counts := make(map[string]int)
	var mu sync.Mutex

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()

			local := make(map[string]int)
			for i := 0; i < keysPerGoroutine; i++ {
				n := string(r.GetNode(fmt.Sprintf("key-%d-%d", gid, i)))
				local[n]++
			}

			mu.Lock()
			for k, v := range local {
				counts[k] += v
			}
			mu.Unlock()
		}(g)
	}
	wg.Wait()

	total := goroutines * keysPerGoroutine
	c1 := float64(counts["n1"]) / float64(total) * 100
	c2 := float64(counts["n2"]) / float64(total) * 100
	c3 := float64(counts["n3"]) / float64(total) * 100

	t.Logf("Concurrent balance n1:%.2f%% n2:%.2f%% n3:%.2f%%", c1, c2, c3)

	// Wide bound — race test, not balance test
	if math.Abs(c1-33.3) > 15 ||
		math.Abs(c2-33.3) > 15 ||
		math.Abs(c3-33.3) > 15 {
		t.Fatalf("concurrent unbalanced: %.2f / %.2f / %.2f", c1, c2, c3)
	}
}

// ---------------- Benchmarks ----------------

// BenchmarkGetNode measures:
// - steady-state lookup latency
// - allocation behavior under read-only access
//
// This represents the hot path in systems like:
// - distributed caches
// - sharded databases
// - request routers
func BenchmarkGetNode(b *testing.B) {
	r := New()
	for i := 0; i < 10; i++ {
		r.AddNode(Node(fmt.Sprintf("n%d", i)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = r.GetNode(fmt.Sprintf("key-%d", i%100_000))
	}
}

// BenchmarkGetNodes measures:
// - cost of replica selection
// - overhead of deduplication across virtual nodes
//
// This models read/write paths in quorum-based systems
// (e.g. N replicas, R/W quorums).
func BenchmarkGetNodes(b *testing.B) {
	r := New()
	for i := 0; i < 10; i++ {
		r.AddNode(Node(fmt.Sprintf("n%d", i)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = r.GetNodes(fmt.Sprintf("key-%d", i%100_000), 3)
	}
}

func unique(nodes []Node) int {
	seen := make(map[Node]struct{})
	for _, n := range nodes {
		seen[n] = struct{}{}
	}
	return len(seen)
}
