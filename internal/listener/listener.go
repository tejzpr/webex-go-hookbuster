/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2025 Tejus Pratap <tejzpr@gmail.com>
 */

package listener

import (
	"fmt"
	"sync"
	"time"

	webex "github.com/tejzpr/webex-go-sdk/v2"
	"github.com/tejzpr/webex-go-sdk/v2/conversation"
	"github.com/tejzpr/webex-go-sdk/v2/device"
	"github.com/tejzpr/webex-go-sdk/v2/mercury"
	"github.com/tejzpr/webex-go-sdk/v2/people"

	"github.com/tejzpr/webex-go-hookbuster/internal/config"
	"github.com/tejzpr/webex-go-hookbuster/internal/display"
	"github.com/tejzpr/webex-go-hookbuster/internal/forwarder"
)

// VerifyAccessToken validates the given access token by calling the
// Webex People API for the authenticated user ("me").
func VerifyAccessToken(accessToken string) (*people.Person, error) {
	client, err := webex.NewClient(accessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Webex client: %w", err)
	}

	person, err := client.People().Get("me")
	if err != nil {
		return nil, fmt.Errorf("the token is not valid or has expired: %w", err)
	}

	return person, nil
}

// Listener manages the Mercury WebSocket connection and event forwarding,
// using the SDK's device, mercury, and conversation clients.
type Listener struct {
	specs              *config.Specs
	client             *webex.WebexClient
	deviceClient       *device.Client
	mercuryClient      *mercury.Client
	conversationClient *conversation.Client
	mu                 sync.Mutex
	running            bool

	// subscriptions tracks which resource/event pairs are active.
	subscriptions map[string]string // resource name -> event filter ("all" or specific)
}

// NewListener creates a new Listener from the given specs.
func NewListener(specs *config.Specs) (*Listener, error) {
	client, err := webex.NewClient(specs.AccessToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Webex client: %w", err)
	}

	return &Listener{
		specs:         specs,
		client:        client,
		subscriptions: make(map[string]string),
	}, nil
}

// Start registers the given resource/event for forwarding, and connects
// Mercury if not already connected.
func (l *Listener) Start(resource *config.Resource, event string) error {
	l.mu.Lock()
	l.subscriptions[resource.Description] = event
	needsConnect := !l.running
	l.mu.Unlock()

	resName := resource.Description

	fmt.Println(display.Info(
		fmt.Sprintf("Listening for events from the %s resource", display.Highlight(resName)),
	))

	// Log individual event handler registrations
	if event == "all" {
		for _, ev := range resource.Events {
			if ev == "all" {
				continue
			}
			fmt.Println(display.Info(
				fmt.Sprintf("Registered handler to forward %s events",
					display.Highlight(fmt.Sprintf("%s:%s", resName, ev))),
			))
		}
	} else {
		fmt.Println(display.Info(
			fmt.Sprintf("Registered handler to forward %s events",
				display.Highlight(fmt.Sprintf("%s:%s", resName, event))),
		))
	}

	if needsConnect {
		return l.connect()
	}
	return nil
}

// connect sets up the SDK internals (device, mercury, conversation) and
// connects to the Mercury WebSocket, following the pattern from the SDK's
// conversation-listen-internal example.
func (l *Listener) connect() error {
	// 1. Device registration (provides the WebSocket URL)
	l.deviceClient = device.New(l.client.Core(), nil)
	fmt.Println(display.Info("Registering device..."))
	if err := l.deviceClient.Register(); err != nil {
		return fmt.Errorf("failed to register device: %w", err)
	}
	fmt.Println(display.Info("Device registered!"))

	deviceInfo := l.deviceClient.GetDevice()
	deviceURL, _ := l.deviceClient.GetDeviceURL()

	// 2. Mercury client (WebSocket transport)
	l.mercuryClient = mercury.New(l.client.Core(), nil)
	l.mercuryClient.SetDeviceProvider(l.deviceClient)

	// 3. Conversation client (parses activities, handles decryption)
	l.conversationClient = conversation.New(l.client.Core(), nil)
	l.conversationClient.SetMercuryClient(l.mercuryClient)
	l.conversationClient.SetEncryptionDeviceInfo(deviceURL, deviceInfo.UserID)

	// 4. Register verb-based handlers that map to hookbuster resources.
	//    The conversation client dispatches conversation.activity events by
	//    verb, so each handler below corresponds to one activity verb.
	l.registerVerbHandlers()

	// 5. Connect Mercury
	fmt.Println(display.Info("Connecting to WebSocket..."))
	if err := l.mercuryClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to Mercury: %w", err)
	}

	l.mu.Lock()
	l.running = true
	l.mu.Unlock()

	fmt.Println(display.Info("Connected to WebSocket!"))
	fmt.Println(display.Info("Webex Initialized"))

	return nil
}

// registerVerbHandlers registers a conversation handler for every verb in
// the VerbToResourceEvent mapping table.
func (l *Listener) registerVerbHandlers() {
	for verb, mapping := range config.VerbToResourceEvent {
		// Capture loop variables for the closure
		v := verb
		m := mapping

		l.conversationClient.On(v, func(activity *conversation.Activity) {
			l.handleActivity(activity, v, m.Resource, m.Event)
		})
	}
}

// handleActivity is called when a conversation activity matches a
// registered verb.  It checks the subscription filter and forwards
// qualifying events to the target.
func (l *Listener) handleActivity(activity *conversation.Activity, verb, resource, event string) {
	// Check if the user is subscribed to this resource
	l.mu.Lock()
	eventFilter, subscribed := l.subscriptions[resource]
	l.mu.Unlock()

	if !subscribed {
		return
	}

	// Check event filter
	if eventFilter != "all" && eventFilter != event {
		return
	}

	// Log the received event
	fmt.Println(display.Info(
		fmt.Sprintf("%s received",
			display.Highlight(fmt.Sprintf("%s:%s", resource, event))),
	))

	// Build the data payload from the activity
	data := buildEventData(activity, verb)

	webhookEvent := config.WebhookEvent{
		Resource:  resource,
		Event:     event,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	// Forward asynchronously so we don't block the Mercury read loop
	target := l.specs.Target
	port := l.specs.Port
	go func() {
		if err := forwarder.Forward(target, port, webhookEvent); err != nil {
			fmt.Println(display.Error(fmt.Sprintf("forward error: %s", err.Error())))
		}
	}()
}

// buildEventData constructs a clean map from a conversation Activity.
func buildEventData(activity *conversation.Activity, verb string) map[string]interface{} {
	data := map[string]interface{}{
		"id":   activity.ID,
		"verb": verb,
	}

	if activity.Actor != nil {
		data["actorId"] = activity.Actor.ID
		data["actorDisplayName"] = activity.Actor.DisplayName
		data["actorEmail"] = activity.Actor.EmailAddress
		data["actorOrgId"] = activity.Actor.OrgID
	}

	if activity.Target != nil {
		data["roomId"] = activity.Target.ID
	}

	if activity.Published != "" {
		data["published"] = activity.Published
	}

	if activity.Content != "" {
		data["content"] = activity.Content
	}

	if activity.Object != nil {
		data["object"] = activity.Object
	}

	// Extract parentId from the raw activity data if present
	if activity.RawData != nil {
		if activityData, ok := activity.RawData["activity"].(map[string]interface{}); ok {
			if parent, ok := activityData["parent"].(map[string]interface{}); ok {
				if parentID, ok := parent["id"].(string); ok {
					data["parentId"] = parentID
				}
			}
		}
	}

	return data
}

// Stop gracefully disconnects the Mercury WebSocket connection.
func (l *Listener) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return nil
	}

	l.running = false

	// Log each resource being stopped
	for resName, eventFilter := range l.subscriptions {
		fmt.Println(display.Info(
			fmt.Sprintf("stopping listener for %s:%s", resName, eventFilter),
		))
	}

	if l.mercuryClient != nil {
		return l.mercuryClient.Disconnect()
	}
	return nil
}
