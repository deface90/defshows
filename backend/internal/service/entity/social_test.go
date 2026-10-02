package entity

import "testing"

func TestSocialTableNames(t *testing.T) {
	if got := (Follow{}).TableName(); got != "follows" {
		t.Errorf("Follow.TableName() = %q, want %q", got, "follows")
	}
	if got := (ActivityEvent{}).TableName(); got != "activity_events" {
		t.Errorf("ActivityEvent.TableName() = %q, want %q", got, "activity_events")
	}
}
