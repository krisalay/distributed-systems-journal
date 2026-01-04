package hlc

import (
	"sync"
	"time"
)

// Config holds clock parameters
type Config struct {
	MaxClockDriftMillis int64 // max uncertainty of local physical clock
}

// Timestamp represents a Hybrid Logical Clock timestamp
type Timestamp struct {
	Physical    int64  // wall-clock time in ms
	Logical     uint16 // logical counter for concurrent events
	Uncertainty int64  // ±ms
}

// Clock implements HLC with bounded uncertainty
type Clock struct {
	mu          sync.Mutex
	physical    int64
	logical     uint16
	uncertainty int64
	cfg         Config
}

// New creates a new HLC instance
func New(cfg Config) *Clock {
	if cfg.MaxClockDriftMillis == 0 {
		cfg.MaxClockDriftMillis = 5 // default 5ms
	}
	return &Clock{cfg: cfg, uncertainty: cfg.MaxClockDriftMillis}
}

// Now returns current HLC timestamp with local uncertainty
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

	// Local uncertainty is at least max drift
	c.uncertainty = c.cfg.MaxClockDriftMillis

	return Timestamp{
		Physical:    c.physical,
		Logical:     c.logical,
		Uncertainty: c.uncertainty,
	}
}

// Update merges a remote timestamp into this clock
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

	// Propagate uncertainty: local + remote + rtt/2
	remoteUncertainty := remote.Uncertainty + rttMillis/2
	c.uncertainty = max(c.uncertainty, remoteUncertainty)
}

// Uncertainty returns current clock uncertainty
func (c *Clock) Uncertainty() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.uncertainty
}

func unixMillis() int64 {
	return time.Now().UnixNano() / 1e6
}

func maxUint16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

// Determines if ts1 definitely happened after ts2
func DefinitelyAfter(ts1, ts2 Timestamp) bool {
	if ts1.Physical > ts2.Physical+ts2.Uncertainty {
		return true
	}
	if ts1.Physical == ts2.Physical && ts1.Logical > ts2.Logical {
		return true
	}
	return false
}
