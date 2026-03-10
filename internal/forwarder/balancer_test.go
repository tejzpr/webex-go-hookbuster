package forwarder

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
)

// helper: create targets from test server URLs
func targetsFromURLs(urls ...string) []config.Target {
	targets := make([]config.Target, len(urls))
	for i, u := range urls {
		targets[i] = config.Target{URL: u}
	}
	return targets
}

// helper: new balancer that we always stop in cleanup
func newTestBalancer(t *testing.T, name string, targets []config.Target) *Balancer {
	t.Helper()
	b := NewBalancer(name, targets)
	t.Cleanup(b.Stop)
	return b
}

func TestBalancer_RoundRobinRotation(t *testing.T) {
	var mu sync.Mutex
	hits := make(map[string]int)

	handler := func(url string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			hits[url]++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		})
	}

	s1 := httptest.NewServer(handler("s1"))
	defer s1.Close()
	s2 := httptest.NewServer(handler("s2"))
	defer s2.Close()
	s3 := httptest.NewServer(handler("s3"))
	defer s3.Close()

	b := newTestBalancer(t, "test", targetsFromURLs(s1.URL, s2.URL, s3.URL))

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	// Send 9 events — should distribute 3 to each
	for i := 0; i < 9; i++ {
		if err := b.Forward(event); err != nil {
			t.Fatalf("Forward() error on iteration %d: %v", i, err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	for _, key := range []string{"s1", "s2", "s3"} {
		if hits[key] != 3 {
			t.Errorf("server %s got %d hits, want 3", key, hits[key])
		}
	}
}

func TestBalancer_SkipsUnhealthyTarget(t *testing.T) {
	var mu sync.Mutex
	hits := make(map[string]int)

	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["s1"]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits["s2"]++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer s2.Close()

	targets := targetsFromURLs(s1.URL, s2.URL)
	b := newTestBalancer(t, "test", targets)

	// Manually mark target 0 unhealthy
	b.targets[0].mu.Lock()
	b.targets[0].healthy = false
	b.targets[0].mu.Unlock()

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	for i := 0; i < 5; i++ {
		if err := b.Forward(event); err != nil {
			t.Fatalf("Forward() error: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if hits["s1"] != 0 {
		t.Errorf("unhealthy s1 got %d hits, want 0", hits["s1"])
	}
	if hits["s2"] != 5 {
		t.Errorf("healthy s2 got %d hits, want 5", hits["s2"])
	}
}

func TestBalancer_RetryOnFailure(t *testing.T) {
	// s1 is unreachable, s2 is healthy
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s2.Close()

	// Use an unreachable URL for s1
	targets := targetsFromURLs("http://127.0.0.1:1", s2.URL)
	b := newTestBalancer(t, "test", targets)

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	// Reset index so first attempt hits s1 (unreachable), should retry to s2
	b.index = atomic.Uint64{}

	err := b.Forward(event)
	if err != nil {
		t.Fatalf("Forward() should succeed via retry, got: %v", err)
	}
}

func TestBalancer_AllTargetsFail(t *testing.T) {
	// Both targets unreachable
	targets := targetsFromURLs("http://127.0.0.1:1", "http://127.0.0.1:2")
	b := newTestBalancer(t, "test", targets)

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	err := b.Forward(event)
	if err == nil {
		t.Fatal("Forward() should return error when all targets fail")
	}
}

func TestBalancer_NoTargets(t *testing.T) {
	b := newTestBalancer(t, "test", []config.Target{})

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	err := b.Forward(event)
	if err == nil {
		t.Fatal("Forward() should return error with no targets")
	}
}

func TestBalancer_TargetBecomesUnhealthyAfterThreshold(t *testing.T) {
	// s1 always fails, s2 always succeeds
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s2.Close()

	targets := targetsFromURLs("http://127.0.0.1:1", s2.URL)
	b := newTestBalancer(t, "test", targets)

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	// Send enough events to trigger the unhealthy threshold on s1
	for i := 0; i < maxConsecutiveFailures+1; i++ {
		b.index = atomic.Uint64{} // force s1 to be attempted first
		_ = b.Forward(event)
	}

	if b.targets[0].isHealthy() {
		t.Error("target s1 should be unhealthy after consecutive failures")
	}
	if !b.targets[1].isHealthy() {
		t.Error("target s2 should still be healthy")
	}
}

func TestBalancer_HealthyCount(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	targets := targetsFromURLs(s1.URL, "http://127.0.0.1:1")
	b := newTestBalancer(t, "test", targets)

	if b.HealthyCount() != 2 {
		t.Errorf("HealthyCount() = %d, want 2", b.HealthyCount())
	}

	// Mark one unhealthy
	b.targets[1].mu.Lock()
	b.targets[1].healthy = false
	b.targets[1].mu.Unlock()

	if b.HealthyCount() != 1 {
		t.Errorf("HealthyCount() = %d, want 1", b.HealthyCount())
	}
}

func TestBalancer_SuccessResetsFailCount(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	targets := targetsFromURLs(s1.URL)
	b := newTestBalancer(t, "test", targets)

	// Simulate some failures without hitting threshold
	b.targets[0].mu.Lock()
	b.targets[0].failCount = maxConsecutiveFailures - 1
	b.targets[0].mu.Unlock()

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	// Successful forward should reset the fail count
	err := b.Forward(event)
	if err != nil {
		t.Fatalf("Forward() error: %v", err)
	}

	b.targets[0].mu.Lock()
	fc := b.targets[0].failCount
	b.targets[0].mu.Unlock()

	if fc != 0 {
		t.Errorf("failCount = %d after success, want 0", fc)
	}
}

func TestBalancer_ProbeRecovery(t *testing.T) {
	// Server that responds to HEAD (health check)
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	targets := targetsFromURLs(s1.URL)
	b := newTestBalancer(t, "test", targets)

	// Mark unhealthy
	b.targets[0].mu.Lock()
	b.targets[0].healthy = false
	b.targets[0].failCount = maxConsecutiveFailures
	b.targets[0].mu.Unlock()

	// Manually run the probe
	client := &http.Client{Timeout: healthCheckTimeout}
	b.probeUnhealthy(client)

	if !b.targets[0].isHealthy() {
		t.Error("target should be healthy after successful probe")
	}
}

func TestBalancer_ProbeDoesNotRecoverUnreachable(t *testing.T) {
	targets := targetsFromURLs("http://127.0.0.1:1")
	b := newTestBalancer(t, "test", targets)

	// Mark unhealthy
	b.targets[0].mu.Lock()
	b.targets[0].healthy = false
	b.targets[0].mu.Unlock()

	client := &http.Client{Timeout: 1 * time.Second}
	b.probeUnhealthy(client)

	if b.targets[0].isHealthy() {
		t.Error("unreachable target should remain unhealthy after probe")
	}
}

func TestBalancer_AllUnhealthyReturnsError(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	targets := targetsFromURLs(s1.URL)
	b := newTestBalancer(t, "test", targets)

	// Mark all unhealthy
	b.targets[0].mu.Lock()
	b.targets[0].healthy = false
	b.targets[0].mu.Unlock()

	event := config.WebhookEvent{Resource: "messages", Event: "created", Data: map[string]interface{}{}}

	err := b.Forward(event)
	if err == nil {
		t.Fatal("Forward() should return error when all targets are unhealthy")
	}
}
