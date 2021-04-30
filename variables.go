package graft

import (
	"time"
)

var (
	// Election timeout MIN and MAX per RAFT spec suggestion.
	MinElectionTimeout = 500 * time.Millisecond
	MaxElectionTimeout = 2 * MinElectionTimeout

	// Heartbeat tick for LEADERS.
	// Should be << MIN_ELECTION_TIMEOUT per RAFT spec.
	HeartbeatInterval = 100 * time.Millisecond
)
