package hashring

import (
	"hash/crc32"
	"sort"
	"strconv"
	"sync"
)

const (
	// DefaultVirtualNodes controls how many virtual points are placed on the ring
	// per unit of node weight.
	//
	// Higher values improve key distribution uniformity at the cost of:
	//   - more memory
	//   - slightly slower ring rebuilds
	//
	// Typical values: 50–200
	DefaultVirtualNodes = 100
)

// Node represents a physical node in the cluster.
// Common examples:
//   - "10.0.0.1:8080"
//   - "cache-a"
//   - "shard-3"
type Node string

// Hasher abstracts the hashing algorithm used by the ring.
//
// Making this pluggable allows swapping CRC32 with faster or higher-quality
// hash functions (e.g. xxhash, murmur3) without changing ring logic.
type Hasher interface {
	Sum32(data []byte) uint32
}

// crc32Hasher is the default hash implementation.
//
// CRC32 is:
//   - fast
//   - deterministic
//   - available in Go stdlib
//
// It is suitable for learning and moderate-scale systems.
type crc32Hasher struct{}

func (c crc32Hasher) Sum32(b []byte) uint32 {
	return crc32.ChecksumIEEE(b)
}

// HashRing implements a weighted consistent hashing ring.
//
// Key properties:
//   - minimal key remapping when nodes are added/removed
//   - support for node weights via virtual nodes
//   - thread-safe lookups and mutations
//
// Internally, the ring is represented as a sorted slice of hash points
// mapping to owning nodes.
type HashRing struct {
	mu sync.RWMutex

	// hasher produces 32-bit hash values for keys and virtual nodes
	hasher Hasher

	// virts is the number of virtual nodes per unit weight
	virts int

	// nodes tracks physical nodes and their weights
	nodes map[Node]int

	// ring holds sorted hash points (virtual nodes)
	ring []uint32

	// nodeMap maps each hash point to its owning physical node
	nodeMap map[uint32]Node
}

// New creates a new HashRing with optional configuration.
//
// By default, it uses:
//   - CRC32 hashing
//   - DefaultVirtualNodes virtual nodes per weight unit
func New(opts ...Option) *HashRing {
	h := &HashRing{
		hasher:  crc32Hasher{},
		virts:   DefaultVirtualNodes,
		nodes:   make(map[Node]int),
		nodeMap: make(map[uint32]Node),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Option configures a HashRing during construction.
type Option func(*HashRing)

// WithHasher replaces the default hash function.
//
// Useful for:
//   - higher throughput (xxhash)
//   - better distribution (murmur3)
//   - experimentation and benchmarking
func WithHasher(h Hasher) Option {
	return func(r *HashRing) {
		r.hasher = h
	}
}

// WithVirtualNodes sets the number of virtual nodes per unit weight.
func WithVirtualNodes(n int) Option {
	return func(r *HashRing) {
		r.virts = n
	}
}

// hash computes the hash value for a given key.
func (h *HashRing) hash(key string) uint32 {
	return h.hasher.Sum32([]byte(key))
}

// AddNode adds a node with default weight = 1.
func (h *HashRing) AddNode(n Node) {
	h.AddNodeWeighted(n, 1)
}

// AddNodeWeighted adds a node with a specified weight.
//
// Weight determines how many virtual nodes are placed on the ring.
// A node with weight 2 receives approximately twice the key space
// of a node with weight 1.
func (h *HashRing) AddNodeWeighted(n Node, weight int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.nodes[n] = weight
	total := h.virts * weight

	// Place virtual nodes on the ring
	for i := 0; i < total; i++ {
		for {
			// Virtual node identity: <node>-<index>
			point := h.hash(string(n) + "-" + strconv.Itoa(i))

			// Avoid hash collisions (rare, but possible)
			if _, exists := h.nodeMap[point]; !exists {
				h.ring = append(h.ring, point)
				h.nodeMap[point] = n
				break
			}

			// Rehash on collision
			i++
		}
	}

	// Keep ring sorted for binary search
	sort.Slice(h.ring, func(i, j int) bool {
		return h.ring[i] < h.ring[j]
	})
}

// RemoveNode removes a node and all its virtual points from the ring.
//
// Only keys owned by this node are remapped, preserving
// the core consistent hashing guarantee.
func (h *HashRing) RemoveNode(n Node) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.nodes, n)

	newRing := make([]uint32, 0, len(h.ring))
	newMap := make(map[uint32]Node)

	for _, p := range h.ring {
		if h.nodeMap[p] != n {
			newRing = append(newRing, p)
			newMap[p] = h.nodeMap[p]
		}
	}

	h.ring = newRing
	h.nodeMap = newMap
}

// GetNode returns the primary node responsible for the given key.
//
// Lookup is performed by hashing the key and selecting the
// first node clockwise on the ring.
func (h *HashRing) GetNode(key string) Node {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 {
		return ""
	}

	point := h.hash(key)
	i := sort.Search(len(h.ring), func(i int) bool {
		return h.ring[i] >= point
	})

	// Wrap around if hash is beyond last point
	if i == len(h.ring) {
		i = 0
	}

	return h.nodeMap[h.ring[i]]
}

// GetNodes returns up to `replicas` distinct nodes for the given key.
//
// Nodes are selected clockwise on the ring, skipping duplicates
// caused by virtual nodes.
//
// This is commonly used for:
//   - replication
//   - quorum systems
//   - multi-node reads/writes
func (h *HashRing) GetNodes(key string, replicas int) []Node {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.ring) == 0 || replicas <= 0 {
		return nil
	}

	// Cannot return more replicas than physical nodes
	max := min(replicas, len(h.nodes))
	nodes := make([]Node, 0, max)

	point := h.hash(key)
	i := sort.Search(len(h.ring), func(j int) bool {
		return h.ring[j] >= point
	})

	// Handle wrap-around
	if i == len(h.ring) {
		i = 0
	}

	seen := make(map[Node]struct{})

	for len(nodes) < max {
		n := h.nodeMap[h.ring[i]]
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			nodes = append(nodes, n)
		}
		i = (i + 1) % len(h.ring)
	}

	return nodes
}
