package entity

import "testing"

func TestSocialTableNames(t *testing.T) {
	if got := (Follow{}).TableName(); got != "follows" {
		t.Errorf("Follow.TableName() = %q, want %q", got, "follows")
	}
	if got := (ActivityEvent{}).TableName(); got != "activity_events" {
		t.Errorf("ActivityEvent.TableName() = %q, want %q", got, "activity_events")
	}
	if got := (Block{}).TableName(); got != "blocks" {
		t.Errorf("Block.TableName() = %q, want %q", got, "blocks")
	}
}

func TestReportTableName(t *testing.T) {
	if got := (Report{}).TableName(); got != "reports" {
		t.Errorf("Report.TableName() = %q, want %q", got, "reports")
	}
}

func TestReportReasonConsts(t *testing.T) {
	cases := map[ReportReason]string{
		ReportSpam:          "spam",
		ReportHarassment:    "harassment",
		ReportInappropriate: "inappropriate",
		ReportOther:         "other",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("ReportReason = %q, want %q", got, want)
		}
	}
}

func TestReportStatusConsts(t *testing.T) {
	cases := map[ReportStatus]string{
		ReportOpen:      "open",
		ReportResolved:  "resolved",
		ReportDismissed: "dismissed",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("ReportStatus = %q, want %q", got, want)
		}
	}
}
