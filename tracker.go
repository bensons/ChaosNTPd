package main

import (
	"math/rand"
	"sync"
	"time"
)

// ClientState tracks the time state for a client
type ClientState struct {
	LastManipulatedTime time.Time
	LastActualTime      time.Time
	FirstSeen           time.Time
	RequestCount        int
}

// ClientTimeTracker tracks manipulated time for each client
type ClientTimeTracker struct {
	mu           sync.RWMutex
	clientStates map[string]*ClientState
	config       *Config
	rand         *rand.Rand
}

// NewClientTimeTracker creates a new client time tracker
func NewClientTimeTracker(config *Config) *ClientTimeTracker {
	tracker := &ClientTimeTracker{
		clientStates: make(map[string]*ClientState),
		config:       config,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Start cleanup goroutine
	go tracker.cleanupLoop()

	return tracker
}

// GetManipulatedTime returns the manipulated time for a client
func (t *ClientTimeTracker) GetManipulatedTime(clientAddr string) (time.Time, float64, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	actualTime := time.Now()
	state, exists := t.clientStates[clientAddr]

	if !exists {
		// Initial request - apply large offset
		offsetMinutes := t.config.TimeManipulation.InitialOffsetMinutes
		offsetSeconds := t.randomFloat(-float64(offsetMinutes*60), float64(offsetMinutes*60))
		manipulatedTime := actualTime.Add(time.Duration(offsetSeconds * float64(time.Second)))

		// Store state
		t.clientStates[clientAddr] = &ClientState{
			LastManipulatedTime: manipulatedTime,
			LastActualTime:      actualTime,
			FirstSeen:           actualTime,
			RequestCount:        1,
		}

		return manipulatedTime, offsetSeconds, true // true = initial request
	}

	// Subsequent request - apply jitter
	elapsed := actualTime.Sub(state.LastActualTime)
	expectedTime := state.LastManipulatedTime.Add(elapsed)

	jitterSeconds := t.config.TimeManipulation.JitterSeconds
	jitter := t.randomFloat(-float64(jitterSeconds), float64(jitterSeconds))
	manipulatedTime := expectedTime.Add(time.Duration(jitter * float64(time.Second)))

	// Update state
	state.LastManipulatedTime = manipulatedTime
	state.LastActualTime = actualTime
	state.RequestCount++

	// Calculate total offset from actual time
	offset := manipulatedTime.Sub(actualTime).Seconds()

	return manipulatedTime, offset, false // false = subsequent request
}

// randomFloat generates a random float between min and max
func (t *ClientTimeTracker) randomFloat(min, max float64) float64 {
	return min + t.rand.Float64()*(max-min)
}

// GetStats returns statistics about tracked clients
func (t *ClientTimeTracker) GetStats() (int, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalClients := len(t.clientStates)
	totalRequests := 0
	for _, state := range t.clientStates {
		totalRequests += state.RequestCount
	}

	return totalClients, totalRequests
}

// cleanupLoop periodically removes stale clients
func (t *ClientTimeTracker) cleanupLoop() {
	interval := time.Duration(t.config.TimeManipulation.ClientTracking.CleanupIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		t.cleanup()
	}
}

// cleanup removes stale clients
func (t *ClientTimeTracker) cleanup() {
	t.mu.Lock()
	defer t.mu.Unlock()

	maxAge := time.Duration(t.config.TimeManipulation.ClientTracking.MaxClientAgeSeconds) * time.Second
	now := time.Now()
	staleCount := 0

	for addr, state := range t.clientStates {
		if now.Sub(state.LastActualTime) > maxAge {
			delete(t.clientStates, addr)
			staleCount++
		}
	}

	if staleCount > 0 {
		LogInfo("Cleaned up %d stale clients, %d remaining", staleCount, len(t.clientStates))
	}

	// Also enforce max clients limit
	maxClients := t.config.TimeManipulation.ClientTracking.MaxTrackedClients
	if len(t.clientStates) > maxClients {
		// Remove oldest clients
		type clientAge struct {
			addr string
			time time.Time
		}
		var clients []clientAge
		for addr, state := range t.clientStates {
			clients = append(clients, clientAge{addr, state.FirstSeen})
		}

		// Sort by age (oldest first) and remove excess
		// For simplicity, just remove first N found
		removeCount := len(t.clientStates) - maxClients
		for i := 0; i < removeCount && i < len(clients); i++ {
			delete(t.clientStates, clients[i].addr)
		}

		LogWarning("Enforced max clients limit, removed %d oldest clients", removeCount)
	}
}
