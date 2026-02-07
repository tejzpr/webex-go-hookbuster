package listener

import (
	"testing"

	"github.com/tejzpr/webex-go-sdk/v2/conversation"
)

func TestBuildEventData_BasicFields(t *testing.T) {
	activity := &conversation.Activity{
		ID:        "msg-123",
		Published: "2026-02-07T02:08:14.939Z",
		Content:   "hello world",
		Actor: &conversation.Actor{
			ID:           "user-1",
			DisplayName:  "Alice",
			EmailAddress: "alice@example.com",
			OrgID:        "org-1",
		},
		Target: &conversation.Target{
			ID: "room-1",
		},
		Object: map[string]interface{}{
			"objectType":  "comment",
			"displayName": "encrypted-content",
		},
	}

	data := buildEventData(activity, "post")

	if data["id"] != "msg-123" {
		t.Errorf("id = %v, want %v", data["id"], "msg-123")
	}
	if data["verb"] != "post" {
		t.Errorf("verb = %v, want %v", data["verb"], "post")
	}
	if data["actorId"] != "user-1" {
		t.Errorf("actorId = %v, want %v", data["actorId"], "user-1")
	}
	if data["actorDisplayName"] != "Alice" {
		t.Errorf("actorDisplayName = %v, want %v", data["actorDisplayName"], "Alice")
	}
	if data["actorEmail"] != "alice@example.com" {
		t.Errorf("actorEmail = %v, want %v", data["actorEmail"], "alice@example.com")
	}
	if data["actorOrgId"] != "org-1" {
		t.Errorf("actorOrgId = %v, want %v", data["actorOrgId"], "org-1")
	}
	if data["roomId"] != "room-1" {
		t.Errorf("roomId = %v, want %v", data["roomId"], "room-1")
	}
	if data["published"] != "2026-02-07T02:08:14.939Z" {
		t.Errorf("published = %v, want %v", data["published"], "2026-02-07T02:08:14.939Z")
	}
	if data["content"] != "hello world" {
		t.Errorf("content = %v, want %v", data["content"], "hello world")
	}
	if data["object"] == nil {
		t.Error("object should not be nil")
	}
}

func TestBuildEventData_NilActor(t *testing.T) {
	activity := &conversation.Activity{
		ID: "msg-456",
	}

	data := buildEventData(activity, "delete")

	if data["id"] != "msg-456" {
		t.Errorf("id = %v, want %v", data["id"], "msg-456")
	}
	if _, ok := data["actorId"]; ok {
		t.Error("actorId should not be present when Actor is nil")
	}
}

func TestBuildEventData_NilTarget(t *testing.T) {
	activity := &conversation.Activity{
		ID: "msg-789",
	}

	data := buildEventData(activity, "post")

	if _, ok := data["roomId"]; ok {
		t.Error("roomId should not be present when Target is nil")
	}
}

func TestBuildEventData_EmptyContent(t *testing.T) {
	activity := &conversation.Activity{
		ID: "msg-000",
	}

	data := buildEventData(activity, "post")

	if _, ok := data["content"]; ok {
		t.Error("content should not be present when Content is empty")
	}
}

func TestBuildEventData_WithParentId(t *testing.T) {
	activity := &conversation.Activity{
		ID: "reply-1",
		RawData: map[string]interface{}{
			"activity": map[string]interface{}{
				"parent": map[string]interface{}{
					"id":   "parent-msg-1",
					"type": "reply",
				},
			},
		},
	}

	data := buildEventData(activity, "post")

	if data["parentId"] != "parent-msg-1" {
		t.Errorf("parentId = %v, want %v", data["parentId"], "parent-msg-1")
	}
}

func TestBuildEventData_NoParentId(t *testing.T) {
	activity := &conversation.Activity{
		ID: "msg-no-parent",
		RawData: map[string]interface{}{
			"activity": map[string]interface{}{
				"verb": "post",
			},
		},
	}

	data := buildEventData(activity, "post")

	if _, ok := data["parentId"]; ok {
		t.Error("parentId should not be present when there is no parent")
	}
}

func TestBuildEventData_NilRawData(t *testing.T) {
	activity := &conversation.Activity{
		ID: "msg-nil-raw",
	}

	data := buildEventData(activity, "share")

	if _, ok := data["parentId"]; ok {
		t.Error("parentId should not be present when RawData is nil")
	}
	// Should still have basic fields
	if data["id"] != "msg-nil-raw" {
		t.Errorf("id = %v, want %v", data["id"], "msg-nil-raw")
	}
	if data["verb"] != "share" {
		t.Errorf("verb = %v, want %v", data["verb"], "share")
	}
}

func TestBuildEventData_MalformedRawData(t *testing.T) {
	// RawData exists but activity key is not a map
	activity := &conversation.Activity{
		ID: "msg-bad-raw",
		RawData: map[string]interface{}{
			"activity": "not-a-map",
		},
	}

	data := buildEventData(activity, "post")

	// Should not panic, parentId should be absent
	if _, ok := data["parentId"]; ok {
		t.Error("parentId should not be present with malformed RawData")
	}
}
