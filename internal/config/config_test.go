package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirehoseResourceNames(t *testing.T) {
	expected := []string{"rooms", "messages", "memberships", "attachmentActions"}

	if len(FirehoseResourceNames) != len(expected) {
		t.Fatalf("expected %d firehose resources, got %d", len(expected), len(FirehoseResourceNames))
	}

	for i, name := range expected {
		if FirehoseResourceNames[i] != name {
			t.Errorf("FirehoseResourceNames[%d] = %q, want %q", i, FirehoseResourceNames[i], name)
		}
	}
}

func TestResourcesExist(t *testing.T) {
	for _, name := range FirehoseResourceNames {
		res, ok := Resources[name]
		if !ok {
			t.Errorf("Resources[%q] not found", name)
			continue
		}
		if res.Description != name {
			t.Errorf("Resources[%q].Description = %q, want %q", name, res.Description, name)
		}
		if res.Alias == "" {
			t.Errorf("Resources[%q].Alias is empty", name)
		}
		if len(res.Events) == 0 {
			t.Errorf("Resources[%q].Events is empty", name)
		}
	}
}

func TestResourceAliasesAreUnique(t *testing.T) {
	seen := make(map[string]string)
	for name, res := range Resources {
		if prev, ok := seen[res.Alias]; ok {
			t.Errorf("duplicate alias %q used by %q and %q", res.Alias, prev, name)
		}
		seen[res.Alias] = name
	}
}

func TestResourceEventsContents(t *testing.T) {
	tests := []struct {
		resource string
		events   []string
	}{
		{"rooms", []string{"all", "created", "updated"}},
		{"messages", []string{"all", "created", "deleted"}},
		{"memberships", []string{"all", "created", "updated", "deleted"}},
		{"attachmentActions", []string{"created"}},
	}

	for _, tt := range tests {
		res := Resources[tt.resource]
		if len(res.Events) != len(tt.events) {
			t.Errorf("Resources[%q].Events length = %d, want %d", tt.resource, len(res.Events), len(tt.events))
			continue
		}
		for i, ev := range tt.events {
			if res.Events[i] != ev {
				t.Errorf("Resources[%q].Events[%d] = %q, want %q", tt.resource, i, res.Events[i], ev)
			}
		}
	}
}

func TestVerbToResourceEventMappings(t *testing.T) {
	tests := []struct {
		verb     string
		resource string
		event    string
	}{
		{"post", "messages", "created"},
		{"share", "messages", "created"},
		{"delete", "messages", "deleted"},
		{"create", "rooms", "created"},
		{"update", "rooms", "updated"},
		{"add", "memberships", "created"},
		{"leave", "memberships", "deleted"},
		{"remove", "memberships", "deleted"},
		{"assignModerator", "memberships", "updated"},
		{"unassignModerator", "memberships", "updated"},
		{"cardAction", "attachmentActions", "created"},
	}

	for _, tt := range tests {
		mapping, ok := VerbToResourceEvent[tt.verb]
		if !ok {
			t.Errorf("VerbToResourceEvent[%q] not found", tt.verb)
			continue
		}
		if mapping.Resource != tt.resource {
			t.Errorf("VerbToResourceEvent[%q].Resource = %q, want %q", tt.verb, mapping.Resource, tt.resource)
		}
		if mapping.Event != tt.event {
			t.Errorf("VerbToResourceEvent[%q].Event = %q, want %q", tt.verb, mapping.Event, tt.event)
		}
	}
}

func TestVerbToResourceEventCoversAllResources(t *testing.T) {
	// Every resource in the firehose list should be reachable from at least one verb
	covered := make(map[string]bool)
	for _, m := range VerbToResourceEvent {
		covered[m.Resource] = true
	}

	for _, name := range FirehoseResourceNames {
		if !covered[name] {
			t.Errorf("resource %q has no verb mapping in VerbToResourceEvent", name)
		}
	}
}

func TestUnknownVerbNotMapped(t *testing.T) {
	if _, ok := VerbToResourceEvent["unknownVerb"]; ok {
		t.Error("unexpected mapping for unknown verb")
	}
}

// ── LoadConfig tests ────────────────────────────────────────────────────

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "hookbuster.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_ValidSinglePipeline(t *testing.T) {
	yaml := `
pipelines:
  - name: "bot"
    token_env: "WEBEX_TOKEN_BOT"
    resources: ["messages", "rooms"]
    events: "all"
    targets:
      - url: "http://localhost:8080"
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if len(cfg.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cfg.Pipelines))
	}
	p := cfg.Pipelines[0]
	if p.Name != "bot" {
		t.Errorf("name = %q, want %q", p.Name, "bot")
	}
	if p.TokenEnv != "WEBEX_TOKEN_BOT" {
		t.Errorf("token_env = %q, want %q", p.TokenEnv, "WEBEX_TOKEN_BOT")
	}
	if len(p.Resources) != 2 {
		t.Errorf("resources count = %d, want 2", len(p.Resources))
	}
	if p.Events != "all" {
		t.Errorf("events = %q, want %q", p.Events, "all")
	}
	if len(p.Targets) != 1 {
		t.Errorf("targets count = %d, want 1", len(p.Targets))
	}
	if p.Targets[0].URL != "http://localhost:8080" {
		t.Errorf("target url = %q, want %q", p.Targets[0].URL, "http://localhost:8080")
	}
}

func TestLoadConfig_MultiplePipelines(t *testing.T) {
	yaml := `
pipelines:
  - name: "bot"
    token_env: "WEBEX_TOKEN_BOT"
    resources: ["messages"]
    events: "all"
    targets:
      - url: "http://localhost:8080"
  - name: "main"
    token_env: "WEBEX_TOKEN_MAIN"
    resources: ["messages", "rooms", "memberships", "attachmentActions"]
    events: "all"
    targets:
      - url: "http://localhost:3000"
      - url: "http://localhost:4000"
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if len(cfg.Pipelines) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(cfg.Pipelines))
	}
	if len(cfg.Pipelines[1].Targets) != 2 {
		t.Errorf("second pipeline targets = %d, want 2", len(cfg.Pipelines[1].Targets))
	}
}

func TestLoadConfig_MissingTokenEnv(t *testing.T) {
	yaml := `
pipelines:
  - name: "bad"
    resources: ["messages"]
    targets:
      - url: "http://localhost:8080"
`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for missing token_env")
	}
	if !strings.Contains(err.Error(), "token_env is required") {
		t.Errorf("error = %q, want it to contain 'token_env is required'", err.Error())
	}
}

func TestLoadConfig_EmptyTargets(t *testing.T) {
	yaml := `
pipelines:
  - name: "no-targets"
    token_env: "WEBEX_TOKEN"
    resources: ["messages"]
    targets: []
`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for empty targets")
	}
	if !strings.Contains(err.Error(), "at least one target is required") {
		t.Errorf("error = %q, want it to contain 'at least one target is required'", err.Error())
	}
}

func TestLoadConfig_EmptyTargetURL(t *testing.T) {
	yaml := `
pipelines:
  - name: "empty-url"
    token_env: "WEBEX_TOKEN"
    resources: ["messages"]
    targets:
      - url: ""
`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for empty target url")
	}
	if !strings.Contains(err.Error(), "target url must not be empty") {
		t.Errorf("error = %q, want it to contain 'target url must not be empty'", err.Error())
	}
}

func TestLoadConfig_InvalidResource(t *testing.T) {
	yaml := `
pipelines:
  - name: "bad-resource"
    token_env: "WEBEX_TOKEN"
    resources: ["messages", "nonexistent"]
    targets:
      - url: "http://localhost:8080"
`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for invalid resource")
	}
	if !strings.Contains(err.Error(), "unknown resource") {
		t.Errorf("error = %q, want it to contain 'unknown resource'", err.Error())
	}
}

func TestLoadConfig_NoPipelines(t *testing.T) {
	yaml := `pipelines: []`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for no pipelines")
	}
	if !strings.Contains(err.Error(), "at least one pipeline") {
		t.Errorf("error = %q, want it to contain 'at least one pipeline'", err.Error())
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/hookbuster.yml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	yaml := `this is: [not valid yaml`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("error = %q, want it to contain 'failed to parse'", err.Error())
	}
}

func TestLoadConfig_EmptyResources(t *testing.T) {
	// Empty resources is valid -- will be treated as firehose (all resources)
	yaml := `
pipelines:
  - name: "no-resources"
    token_env: "WEBEX_TOKEN"
    resources: []
    targets:
      - url: "http://localhost:8080"
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig() returned error: %v", err)
	}
	if len(cfg.Pipelines[0].Resources) != 0 {
		t.Errorf("resources count = %d, want 0", len(cfg.Pipelines[0].Resources))
	}
}
