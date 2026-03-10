/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package forwarder

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
	"github.com/tejzpr/webex-go-hookbuster/internal/display"
)

const (
	// maxConsecutiveFailures is the number of consecutive forward failures
	// before a target is marked unhealthy.
	maxConsecutiveFailures = 3

	// healthCheckInterval is how often the background goroutine probes
	// unhealthy targets.
	healthCheckInterval = 10 * time.Second

	// healthCheckTimeout is the HTTP timeout for a single health probe.
	healthCheckTimeout = 5 * time.Second
)

// targetState tracks the health of a single forwarding target.
type targetState struct {
	mu        sync.Mutex
	url       string
	healthy   bool
	failCount int
}

// markFailed increments the failure counter and marks the target unhealthy
// once the threshold is reached. Returns true if the target just became unhealthy.
func (ts *targetState) markFailed() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.failCount++
	if ts.healthy && ts.failCount >= maxConsecutiveFailures {
		ts.healthy = false
		return true
	}
	return false
}

// markHealthy resets the failure counter and marks the target healthy.
// Returns true if the target was previously unhealthy.
func (ts *targetState) markHealthy() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	wasUnhealthy := !ts.healthy
	ts.healthy = true
	ts.failCount = 0
	return wasUnhealthy
}

// isHealthy returns the current health status.
func (ts *targetState) isHealthy() bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.healthy
}

// Balancer implements round-robin forwarding with health checks and retry.
type Balancer struct {
	targets []*targetState
	index   atomic.Uint64
	stopCh  chan struct{}
	wg      sync.WaitGroup
	name    string // pipeline name for logging
}

// NewBalancer creates a round-robin balancer for the given targets and starts
// the background health-check goroutine.
func NewBalancer(name string, targets []config.Target) *Balancer {
	states := make([]*targetState, len(targets))
	for i, t := range targets {
		states[i] = &targetState{
			url:     t.URL,
			healthy: true,
		}
	}

	b := &Balancer{
		targets: states,
		stopCh:  make(chan struct{}),
		name:    name,
	}

	b.wg.Add(1)
	go b.healthCheckLoop()

	return b
}

// Forward sends the event to the next healthy target using round-robin.
// On failure it retries up to len(targets)-1 more healthy targets.
// Returns an error only if all targets fail or none are healthy.
func (b *Balancer) Forward(event config.WebhookEvent) error {
	n := uint64(len(b.targets))
	if n == 0 {
		return fmt.Errorf("no targets configured")
	}

	startIdx := b.index.Add(1) - 1
	var lastErr error

	for attempt := uint64(0); attempt < n; attempt++ {
		idx := (startIdx + attempt) % n
		ts := b.targets[idx]

		if !ts.isHealthy() {
			continue
		}

		err := ForwardToURL(ts.url, event)
		if err == nil {
			ts.markHealthy()
			return nil
		}

		lastErr = err
		becameUnhealthy := ts.markFailed()
		if becameUnhealthy {
			fmt.Println(display.Error(fmt.Sprintf("[%s] target %s marked unhealthy after %d consecutive failures",
				b.name, ts.url, maxConsecutiveFailures)))
		}
	}

	if lastErr != nil {
		return fmt.Errorf("all targets failed, last error: %w", lastErr)
	}
	return fmt.Errorf("no healthy targets available")
}

// Stop shuts down the background health-check goroutine.
func (b *Balancer) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

// healthCheckLoop periodically probes unhealthy targets and marks them
// healthy again when they respond.
func (b *Balancer) healthCheckLoop() {
	defer b.wg.Done()

	client := &http.Client{Timeout: healthCheckTimeout}
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.probeUnhealthy(client)
		}
	}
}

// probeUnhealthy sends a HEAD request to each unhealthy target. If the
// target responds (any status code), it is marked healthy again.
func (b *Balancer) probeUnhealthy(client *http.Client) {
	for _, ts := range b.targets {
		if ts.isHealthy() {
			continue
		}

		resp, err := client.Head(ts.url)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// Any response means the target is reachable again
		if ts.markHealthy() {
			fmt.Println(display.Info(fmt.Sprintf("[%s] target %s recovered, marked healthy",
				b.name, ts.url)))
		}
	}
}

// HealthyCount returns the number of currently healthy targets.
// Useful for diagnostics and testing.
func (b *Balancer) HealthyCount() int {
	count := 0
	for _, ts := range b.targets {
		if ts.isHealthy() {
			count++
		}
	}
	return count
}
