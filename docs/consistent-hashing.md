# Consistent Hashing

Consistent hashing is a technique used in distributed systems to map keys (e.g., cache keys, object IDs, user IDs) to nodes in a cluster in a way that **minimizes remapping** when nodes are added or removed.

---

## The Problem With Modulo Hashing

A naïve approach to sharding is:

```text
node = hash(key) % N
```

Where `N` is the number of nodes.

**Problem:** when `N` changes, *almost all keys move*.

This causes:

* cache stampedes
* massive data reshuffling
* downtime during scaling events

---

## Core Idea of Consistent Hashing

Instead of mapping keys directly to nodes, we:

1. Map both **nodes and keys** into the same hash space
2. Arrange them on a **logical ring** (usually 0 → 2³²)
3. Assign each key to the **next node clockwise** on the ring

When a node joins or leaves, only the keys in its immediate neighborhood are affected.

---

## Hash Ring Visualization

```
0 ------------------------------------------------------> 2^32
|                                                         |
|   k1        n1        k2     n2           k3       n3   |
|                                                         |
 ---------------------------------------------------------
```

Each `nX` is a node placed on the ring using a hash of its identity.
Each `kX` is a key hashed into the same space.

A key is owned by the **first node clockwise** from its position.

---

## Virtual Nodes (VNodes)

Real systems do **not** place just one point per node.

Instead, each physical node is assigned many *virtual nodes*:

```
Physical node A → A-0, A-1, A-2, ... A-99
```

### Why virtual nodes matter

* Smooths out uneven hash distribution
* Prevents "hot" nodes
* Allows **weighted capacity**

A node with weight 2 simply gets twice as many virtual nodes.

---

## Minimal Remapping Property

When a node is added:

* Only keys that now fall before that node move

When a node is removed:

* Only keys owned by that node are reassigned

This property is what makes consistent hashing ideal for:

* distributed caches (Redis, Memcached)
* object stores
* sharded databases
* leaderless replication systems

---

## Replication Using the Ring

Replication is implemented by walking the ring clockwise:

```
Primary → Replica 1 → Replica 2 → ...
```

Virtual-node duplicates are skipped to ensure **distinct physical nodes**.

This enables:

* quorum reads/writes
* fault tolerance
* multi-region placement

---

## Hash Function Choice

Good hash functions should have:

* strong avalanche properties
* uniform distribution
* deterministic output

Common choices:

| Hash    | Notes                                    |
| ------- | ---------------------------------------- |
| CRC32   | Fast, simple, OK for small clusters      |
| Murmur3 | Better distribution, common in databases |
| xxHash  | Extremely fast, good quality             |

This implementation allows swapping hash functions via configuration.

---

## Trade-offs

| Benefit               | Cost                              |
| --------------------- | --------------------------------- |
| Minimal remapping     | Extra memory for virtual nodes    |
| Easy scaling          | Ring rebuild on membership change |
| Deterministic routing | Care needed around collisions     |

---

## When NOT to Use Consistent Hashing

Consistent hashing is not ideal when:

* strict ordering is required
* frequent rebalancing is acceptable
* global coordination already exists

In such cases, range partitioning or centralized metadata may be simpler.

---

## Summary

Consistent hashing trades a small amount of memory and complexity for:

* excellent scalability
* predictable behavior
* minimal disruption during topology changes

That trade-off is why it underpins many real-world distributed systems.
