# Distributed Systems Journal

A hands-on, evolving journal of **distributed systems concepts**, implemented in Go.

This repository is not a framework or a black-box library. It is a **learning-oriented, production-minded** collection of implementations that explain *why systems are built the way they are*, not just *how*.

Each module is:

* small and focused
* heavily documented
* benchmarked
* safe for real-world usage with clear trade-offs

---

## 📦 Modules

### 1. Consistent Hash Ring (`hashring`)

A production-quality implementation of **consistent hashing** with:

* virtual nodes (vnodes)
* weighted capacity
* minimal key remapping
* replica selection
* pluggable hash functions
* concurrency safety

This is the foundational primitive behind:

* distributed caches (Redis, Memcached)
* sharded databases
* object storage systems
* leaderless replication

📄 **Deep dive:** [`docs/consistent-hashing.md`](docs/consistent-hashing.md)

---

## 🧠 How Consistent Hashing Works

Instead of mapping keys directly to nodes using modulo arithmetic, both **keys and nodes** are mapped into the same logical hash space and arranged on a ring.

```
0 ------------------------------------------------------> 2^32
|                                                         |
|   k1        n1        k2     n2           k3       n3   |
|                                                         |
 ---------------------------------------------------------
```

* `n1`, `n2`, `n3` are **nodes** placed on the ring
* `k1`, `k2`, `k3` are **keys**
* A key is owned by the **first node clockwise** on the ring

### Virtual Nodes

Each physical node is represented by many virtual points:

```
n1 → n1-0, n1-1, n1-2, ...
```

This:

* smooths hash distribution
* avoids hot spots
* enables weighted capacity

---

## ✨ Features (HashRing)

* ✅ Minimal key remapping on node add/remove
* ⚖️ Weighted nodes via virtual nodes
* 🔁 Replica selection with deduplication
* 🔒 Thread-safe lookups and updates
* 🔌 Pluggable hash functions
* 📈 Benchmarked and race-tested

---

## 🚀 Usage Example

```go
ring := hashring.New(
    hashring.WithVirtualNodes(100),
)

ring.AddNode("node-a")
ring.AddNodeWeighted("node-b", 2)

primary := ring.GetNode("user:123")
replicas := ring.GetNodes("user:123", 3)
```

---

## 🔒 Concurrency Model

This implementation uses `sync.RWMutex`:

* **Writes** (adding/removing nodes) take an exclusive lock
* **Reads** (key lookups) take a shared lock

### Why this design?

* Membership changes are rare
* Lookups are extremely frequent
* RWMutex provides excellent performance for this ratio
* Simpler, safer, and easier to reason about

### Immutable Ring Alternative (Advanced)

High-throughput systems often use immutable rings:

1. Build a new ring off-thread
2. Atomically swap a pointer (`atomic.Value`)
3. Reads become completely lock-free

**Trade-off:** higher complexity in exchange for zero read contention.

This repo intentionally favors **clarity and correctness** over premature optimization.

---

## 📊 Benchmarks

Benchmarks are designed to validate **real-world properties**, not just speed.

* `BenchmarkGetNode`

  * Measures steady-state lookup latency
  * Represents the hot path in caches and routers

* `BenchmarkGetNodes`

  * Measures replica selection cost
  * Models quorum-based read/write paths

Run them with:

```bash
go test -bench=. -benchmem ./...
```

---

## 🧪 Testing Philosophy

Tests focus on **behavioral guarantees**, not internal details:

* balance and distribution
* weighted capacity correctness
* replica uniqueness
* race safety under concurrency

Run all tests:

```bash
go test -v -race ./...
```

---

## 🎯 Goals of This Repository

* Build intuition for distributed systems primitives
* Bridge theory with production-grade Go code
* Document real trade-offs, not just happy paths
* Serve as a long-term reference and learning journal

This repo is intentionally incremental — each module builds toward more complex systems.

---

## 🛣️ Roadmap

Planned additions:

* Jump Consistent Hash
* Lock-free / immutable hash rings
* Failure-aware replica placement
* Membership backed by etcd
* Distributed cache built on top of the ring

---

## 🤝 Contributions

This is a learning-focused repository.

* Ideas, discussions, and improvements are welcome
* Code should prioritize clarity and correctness
* Documentation is valued as much as implementation

---

## 📜 License

MIT License

---

> *Distributed systems are best learned by building them — carefully, incrementally, and honestly.*
