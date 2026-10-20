package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReminderNeeded(t *testing.T) {
	now := time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		due   time.Time
		phone string
		want  bool
	}{{"tomorrow", now.Add(12 * time.Hour), "+15550001", true}, {"later", now.Add(48 * time.Hour), "+15550001", false}, {"missing phone", now.Add(12 * time.Hour), "", false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reminderNeeded(CourseDeadline{DueAt: tt.due, Phone: tt.phone}, now)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSMSClientSendRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(payload) != 2 || payload["to"] != "+15550001" || payload["body"] != "deadline reminder" {
			t.Errorf("unexpected SMS payload: %v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"message_id":"test-id"}}`))
	}))
	defer server.Close()

	id, err := (SMSClient{BaseURL: server.URL, Key: "test-key", HTTP: server.Client()}).Send("+15550001", "deadline reminder", "test-request")
	if err != nil || id != "test-id" {
		t.Fatalf("Send() = %q, %v; want test-id, nil", id, err)
	}
}
