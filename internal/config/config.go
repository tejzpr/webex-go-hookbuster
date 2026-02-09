/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Resource describes a Webex resource that can emit events.
type Resource struct {
	Alias       string   // Short alias for CLI selection (e.g. "r", "m")
	Description string   // Full resource name (e.g. "rooms", "messages")
	Events      []string // Supported event types including "all"
}

// Selection holds the user's chosen resource and event.
type Selection struct {
	Resource string
	Event    string
}

// Specs holds the complete configuration for a hookbuster session.
type Specs struct {
	Target      string
	AccessToken string
	Port        int
	Selection   Selection
}

// WebhookEvent is the payload forwarded to the target application.
type WebhookEvent struct {
	Resource  string      `json:"resource"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// FirehoseResourceNames lists all resource names for "all" (firehose) mode.
var FirehoseResourceNames = []string{"rooms", "messages", "memberships", "attachmentActions"}

// Resources maps resource names to their definitions.
// This mirrors the Node.js hookbuster's cli.js options object.
var Resources = map[string]*Resource{
	"rooms": {
		Alias:       "r",
		Description: "rooms",
		Events:      []string{"all", "created", "updated"},
	},
	"messages": {
		Alias:       "m",
		Description: "messages",
		Events:      []string{"all", "created", "deleted"},
	},
	"memberships": {
		Alias:       "mm",
		Description: "memberships",
		Events:      []string{"all", "created", "updated", "deleted"},
	},
	"attachmentActions": {
		Alias:       "aa",
		Description: "attachmentActions",
		Events:      []string{"created"},
	},
}

// ── Multi-pipeline configuration types ──────────────────────────────────

// Target represents a single webhook forwarding destination.
type Target struct {
	URL string `yaml:"url" json:"url"`
}

// Pipeline represents a single token-to-targets mapping.
type Pipeline struct {
	Name      string   `yaml:"name"      json:"name"`
	TokenEnv  string   `yaml:"token_env" json:"token_env"`
	Resources []string `yaml:"resources" json:"resources"`
	Events    string   `yaml:"events"    json:"events"`
	Targets   []Target `yaml:"targets"   json:"targets"`
}

// HookbusterConfig is the top-level YAML configuration for multi-pipeline mode.
type HookbusterConfig struct {
	Pipelines []Pipeline `yaml:"pipelines" json:"pipelines"`
}

// LoadConfig reads and validates a YAML configuration file.
func LoadConfig(path string) (*HookbusterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg HookbusterConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.Pipelines) == 0 {
		return nil, fmt.Errorf("config must contain at least one pipeline")
	}

	for i, p := range cfg.Pipelines {
		if err := validatePipeline(i, p); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

// validatePipeline checks a single pipeline for required fields and valid values.
func validatePipeline(index int, p Pipeline) error {
	if p.TokenEnv == "" {
		return fmt.Errorf("pipeline %d (%q): token_env is required", index, p.Name)
	}
	if len(p.Targets) == 0 {
		return fmt.Errorf("pipeline %d (%q): at least one target is required", index, p.Name)
	}
	for _, t := range p.Targets {
		if t.URL == "" {
			return fmt.Errorf("pipeline %d (%q): target url must not be empty", index, p.Name)
		}
	}
	for _, r := range p.Resources {
		if _, ok := Resources[r]; !ok {
			return fmt.Errorf("pipeline %d (%q): unknown resource %q", index, p.Name, r)
		}
	}
	return nil
}

// VerbToResourceEvent maps Mercury activity verbs (and object types where
// needed) to hookbuster resource/event pairs.
//
// Mercury delivers all real-time events as "conversation.activity" with a
// verb field.  This table lets the listener translate those into the
// resource + event pairs that the original Node.js hookbuster uses.
var VerbToResourceEvent = map[string]struct {
	Resource string
	Event    string
}{
	"post":              {Resource: "messages", Event: "created"},
	"share":             {Resource: "messages", Event: "created"},
	"delete":            {Resource: "messages", Event: "deleted"},
	"create":            {Resource: "rooms", Event: "created"},
	"update":            {Resource: "rooms", Event: "updated"},
	"add":               {Resource: "memberships", Event: "created"},
	"leave":             {Resource: "memberships", Event: "deleted"},
	"remove":            {Resource: "memberships", Event: "deleted"},
	"assignModerator":   {Resource: "memberships", Event: "updated"},
	"unassignModerator": {Resource: "memberships", Event: "updated"},
	"cardAction":        {Resource: "attachmentActions", Event: "created"},
}
