package hlc

import (
	"sync"
	"time"
)

// Config configures a Clock instance.
//
// MaxClockDriftMillis bounds the expected error of the local physical clock,
// expressed in milliseconds. It is used as the minimum uncertainty attached
// to timestamps produced by the clock.
type Config struct {
	MaxClockDriftMillis int64 // Maximum tolerated drift of the local clock in milliseconds.
}

// Timestamp represents a Hybrid Logical Clock timestamp with bounded uncertainty.
//
// Physical is the wall-clock component in milliseconds since Unix epoch.
// Logical is a counter that disambiguates events when physical time does not advance.
// Uncertainty is a symmetric error bound (±milliseconds) around the physical value.
type Timestamp struct {
	Physical    int64  // Wall-clock time in milliseconds.
	Logical     uint16 // Logical counter for concurrent or ambiguous events.
	Uncertainty int64  // Symmetric uncertainty bound in milliseconds.
}

// Clock maintains Hybrid Logical Clock state with bounded uncertainty.
//
// A Clock is safe for concurrent use by multiple goroutines. It should typically
// be instantiated once per process and used to stamp all local events.
type Clock struct {
	mu          sync.Mutex
	physical    int64
	logical     uint16
	uncertainty int64
	cfg         Config
}

// New returns a new Clock configured with cfg.
//
// If MaxClockDriftMillis is zero, New applies a default of 5 milliseconds.
// The returned Clock starts with physical time set to zero and uncertainty
// equal to cfg.MaxClockDriftMillis.
func New(cfg Config) *Clock {
	if cfg.MaxClockDriftMillis == 0 {
		cfg.MaxClockDriftMillis = 5 // Default to 5 ms drift if unspecified.
	}
	return &Clock{cfg: cfg, uncertainty: cfg.MaxClockDriftMillis}
}

// Now returns a new Timestamp representing the current local HLC time.
//
// Now observes the local wall clock, advances the physical component
// monotonically, and increments the logical component when the physical
// clock does not move forward. The returned uncertainty is at least the
// configured MaxClockDriftMillis.
func (c *Clock) Now() Timestamp {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := unixMillis()
	if now > c.physical {
		c.physical = now
		c.logical = 0
	} else {
		c.logical++
	}

	// Local uncertainty is at least the configured maximum drift.
	c.uncertainty = c.cfg.MaxClockDriftMillis

	return Timestamp{
		Physical:    c.physical,
		Logical:     c.logical,
		Uncertainty: c.uncertainty,
	}
}

// Update incorporates a remote timestamp into the local clock state.
//
// remote is a timestamp received from another node, and rttMillis is the
// estimated round-trip time in milliseconds between nodes. Update advances
// the local physical and logical components to preserve causality and
// propagates uncertainty by accounting for remote.Uncertainty and half the RTT.
func (c *Clock) Update(remote Timestamp, rttMillis int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := unixMillis()
	maxPhysical := max(c.physical, max(remote.Physical, now))

	switch {
	case maxPhysical == c.physical && maxPhysical == remote.Physical:
		c.logical = maxUint16(c.logical, remote.Logical) + 1
	case maxPhysical == c.physical:
		c.logical++
	case maxPhysical == remote.Physical:
		c.logical = remote.Logical + 1
	default:
		c.logical = 0
	}

	c.physical = maxPhysical

	// Propagate uncertainty: take the maximum of local uncertainty and
	// the remote uncertainty extended by half the observed RTT.
	remoteUncertainty := remote.Uncertainty + rttMillis/2
	c.uncertainty = max(c.uncertainty, remoteUncertainty)
}

// Uncertainty returns the current uncertainty bound of the clock in milliseconds.
//
// The returned value reflects the maximum of the local drift configuration and
// any remote uncertainty observed through Update calls.
func (c *Clock) Uncertainty() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.uncertainty
}

// unixMillis returns the current wall-clock time in milliseconds since Unix epoch.
func unixMillis() int64 {
	return time.Now().UnixNano() / 1e6
}

// maxUint16 returns the larger of a and b.
func maxUint16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// DefinitelyAfter reports whether ts1 is guaranteed to have occurred after ts2.
//
// This holds only if the earliest possible time for ts1 is strictly greater
// than the latest possible time for ts2, given their uncertainty bounds.
// When DefinitelyAfter returns false, the relative ordering of ts1 and ts2
// is ambiguous and must not be treated as strictly ordered.
func DefinitelyAfter(ts1, ts2 Timestamp) bool {
	if ts1.Physical > ts2.Physical+ts2.Uncertainty {
		return true
	}
	if ts1.Physical == ts2.Physical && ts1.Logical > ts2.Logical {
		return true
	}
	return false
}
