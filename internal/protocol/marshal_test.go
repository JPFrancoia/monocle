package protocol

import (
	"testing"
)

func TestEncodeDecodeSubscribe(t *testing.T) {
	msg := &SubscribeMsg{
		Type:       TypeSubscribe,
		Events:     []string{"feedback_submitted", "pause_changed"},
		ClientKind: SubscribeClientFrontend,
	}

	data, err := Encode(msg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := Decode(data[:len(data)-1]) // strip newline
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	sub, ok := decoded.(*SubscribeMsg)
	if !ok {
		t.Fatalf("expected *SubscribeMsg, got %T", decoded)
	}
	if sub.Type != TypeSubscribe {
		t.Errorf("type = %q, want %q", sub.Type, TypeSubscribe)
	}
	if len(sub.Events) != 2 {
		t.Errorf("events count = %d, want 2", len(sub.Events))
	}
	if sub.ClientKind != SubscribeClientFrontend {
		t.Errorf("client kind = %q, want %q", sub.ClientKind, SubscribeClientFrontend)
	}
}

func TestEncodeDecodeSubscribeResponse(t *testing.T) {
	msg := &SubscribeResponse{
		Type:    TypeSubscribeResponse,
		Success: true,
	}

	data, err := Encode(msg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := Decode(data[:len(data)-1])
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	resp, ok := decoded.(*SubscribeResponse)
	if !ok {
		t.Fatalf("expected *SubscribeResponse, got %T", decoded)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
}

func TestEncodeDecodeEventNotification(t *testing.T) {
	msg := &EventNotification{
		Type:  TypeEventNotification,
		Event: "feedback_submitted",
		Payload: map[string]any{
			"message": "## Review — Changes Requested",
			"status":  "request_changes",
		},
	}

	data, err := Encode(msg)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := Decode(data[:len(data)-1])
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	notif, ok := decoded.(*EventNotification)
	if !ok {
		t.Fatalf("expected *EventNotification, got %T", decoded)
	}
	if notif.Event != "feedback_submitted" {
		t.Errorf("event = %q, want %q", notif.Event, "feedback_submitted")
	}
	if notif.Payload["message"] != "## Review — Changes Requested" {
		t.Errorf("payload.message = %q", notif.Payload["message"])
	}
}

func TestDecodeUnknownType(t *testing.T) {
	_, err := Decode([]byte(`{"type":"unknown_type"}`))
	if err == nil {
		t.Error("expected error for unknown type")
	}
}
