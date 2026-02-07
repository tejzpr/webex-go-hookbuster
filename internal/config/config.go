/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package config

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
