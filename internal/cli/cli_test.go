package cli

import (
	"bufio"
	"strings"
	"testing"
)

// setInput replaces the package-level scanner with one reading from s.
func setInput(s string) {
	scanner = bufio.NewScanner(strings.NewReader(s))
}

// ── RequestToken tests ──────────────────────────────────────────────────

func TestRequestToken_Valid(t *testing.T) {
	setInput("my-secret-token\n")
	token, err := RequestToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("token = %q, want %q", token, "my-secret-token")
	}
}

func TestRequestToken_Empty(t *testing.T) {
	setInput("\n")
	_, err := RequestToken()
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if err.Error() != "token empty" {
		t.Errorf("error = %q, want %q", err.Error(), "token empty")
	}
}

func TestRequestToken_TrimSpaces(t *testing.T) {
	setInput("  spaced-token  \n")
	token, err := RequestToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "spaced-token" {
		t.Errorf("token = %q, want %q", token, "spaced-token")
	}
}

// ── RequestTarget tests ─────────────────────────────────────────────────

func TestRequestTarget_Valid(t *testing.T) {
	setInput("localhost\n")
	target, err := RequestTarget()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target != "localhost" {
		t.Errorf("target = %q, want %q", target, "localhost")
	}
}

func TestRequestTarget_Empty(t *testing.T) {
	setInput("\n")
	_, err := RequestTarget()
	if err == nil {
		t.Fatal("expected error for empty target")
	}
}

// ── RequestPort tests ───────────────────────────────────────────────────

func TestRequestPort_Valid(t *testing.T) {
	setInput("8080\n")
	port, err := RequestPort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8080 {
		t.Errorf("port = %d, want %d", port, 8080)
	}
}

func TestRequestPort_Empty(t *testing.T) {
	setInput("\n")
	_, err := RequestPort()
	if err == nil {
		t.Fatal("expected error for empty port")
	}
}

func TestRequestPort_NotANumber(t *testing.T) {
	setInput("abc\n")
	_, err := RequestPort()
	if err == nil {
		t.Fatal("expected error for non-numeric port")
	}
	if err.Error() != "not a number" {
		t.Errorf("error = %q, want %q", err.Error(), "not a number")
	}
}

// ── RequestResource tests ───────────────────────────────────────────────

func TestRequestResource_All(t *testing.T) {
	setInput("a\n")
	result, err := RequestResource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.AllResources {
		t.Error("expected AllResources to be true")
	}
}

func TestRequestResource_Rooms(t *testing.T) {
	setInput("r\n")
	result, err := RequestResource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AllResources {
		t.Error("expected AllResources to be false")
	}
	if result.Single == nil || result.Single.Description != "rooms" {
		t.Errorf("expected rooms resource, got %+v", result.Single)
	}
}

func TestRequestResource_Messages(t *testing.T) {
	setInput("m\n")
	result, err := RequestResource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Single == nil || result.Single.Description != "messages" {
		t.Errorf("expected messages resource, got %+v", result.Single)
	}
}

func TestRequestResource_Memberships(t *testing.T) {
	setInput("mm\n")
	result, err := RequestResource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Single == nil || result.Single.Description != "memberships" {
		t.Errorf("expected memberships resource, got %+v", result.Single)
	}
}

func TestRequestResource_AttachmentActions(t *testing.T) {
	setInput("aa\n")
	result, err := RequestResource()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Single == nil || result.Single.Description != "attachmentActions" {
		t.Errorf("expected attachmentActions resource, got %+v", result.Single)
	}
}

func TestRequestResource_Invalid(t *testing.T) {
	setInput("xyz\n")
	_, err := RequestResource()
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
	if err.Error() != "invalid selection" {
		t.Errorf("error = %q, want %q", err.Error(), "invalid selection")
	}
}

func TestRequestResource_Empty(t *testing.T) {
	setInput("\n")
	_, err := RequestResource()
	if err == nil {
		t.Fatal("expected error for empty selection")
	}
}

// ── RequestEvent tests ──────────────────────────────────────────────────

func TestRequestEvent_All(t *testing.T) {
	setInput("a\n")
	event, err := RequestEvent([]string{"all", "created", "updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != "all" {
		t.Errorf("event = %q, want %q", event, "all")
	}
}

func TestRequestEvent_Created(t *testing.T) {
	setInput("c\n")
	event, err := RequestEvent([]string{"all", "created", "deleted"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event != "created" {
		t.Errorf("event = %q, want %q", event, "created")
	}
}

func TestRequestEvent_Invalid(t *testing.T) {
	setInput("z\n")
	_, err := RequestEvent([]string{"all", "created"})
	if err == nil {
		t.Fatal("expected error for invalid event")
	}
	if err.Error() != "event invalid" {
		t.Errorf("error = %q, want %q", err.Error(), "event invalid")
	}
}

func TestRequestEvent_Empty(t *testing.T) {
	setInput("\n")
	_, err := RequestEvent([]string{"all", "created"})
	if err == nil {
		t.Fatal("expected error for empty event")
	}
}

// ── EOF handling ────────────────────────────────────────────────────────

func TestRequestToken_EOF(t *testing.T) {
	setInput("") // empty = immediate EOF
	_, err := RequestToken()
	if err == nil {
		t.Fatal("expected error on EOF")
	}
}
