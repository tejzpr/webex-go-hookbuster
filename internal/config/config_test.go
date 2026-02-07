package config

import (
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
