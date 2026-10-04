import type { ReportReason } from '@/shared/api/admin/model'

/**
 * Canonical Russian labels for report reasons, shared across every surface
 * (user-facing ReportModal + admin ReportsPage) so the wording stays in sync.
 * The admin `ReportReason` and social `ReportRequestReason` enums carry the same
 * string values, so this single map keys both.
 */
export const reportReasonLabels: Record<ReportReason, string> = {
  spam: 'Спам',
  harassment: 'Оскорбления или травля',
  inappropriate: 'Неприемлемый контент',
  other: 'Другое',
}
