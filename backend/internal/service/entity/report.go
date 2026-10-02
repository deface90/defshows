package entity

import "time"

// ReportReason enumerates the allowed reasons a user can report another user.
type ReportReason string

const (
	// ReportSpam flags spam / unsolicited promotional content.
	ReportSpam ReportReason = "spam"
	// ReportHarassment flags harassment or abusive behaviour.
	ReportHarassment ReportReason = "harassment"
	// ReportInappropriate flags inappropriate or offensive content.
	ReportInappropriate ReportReason = "inappropriate"
	// ReportOther is a catch-all reason with free-text detail in the note.
	ReportOther ReportReason = "other"
)

// ReportStatus is the moderation lifecycle state of a report.
type ReportStatus string

const (
	// ReportOpen is an unreviewed report awaiting admin action.
	ReportOpen ReportStatus = "open"
	// ReportResolved marks a report the admin acted upon.
	ReportResolved ReportStatus = "resolved"
	// ReportDismissed marks a report the admin reviewed and took no action on.
	ReportDismissed ReportStatus = "dismissed"
)

// Report is a user-filed moderation report against another user, persisted to a
// durable queue for admin review (App Store Guideline 1.2 requires acting ≤24h).
type Report struct {
	ID           int64 `gorm:"primaryKey"`
	ReporterID   int64
	TargetUserID int64
	Reason       ReportReason
	Note         string
	Status       ReportStatus
	CreatedAt    time.Time
	ResolvedAt   *time.Time
	ResolvedBy   *int64
}

// TableName maps Report to the reports table.
func (Report) TableName() string { return "reports" }
