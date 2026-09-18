import { UserShowStatus } from '@/shared/api/tracking/model'

// Labels derived from the generated status enum (kept in sync with the contract).
export const STATUS_LABELS: Record<string, string> = {
  [UserShowStatus.watching]: 'Смотрю',
  [UserShowStatus.plan_to_watch]: 'В планах',
  [UserShowStatus.on_hold]: 'На паузе',
  [UserShowStatus.completed]: 'Просмотрено',
  [UserShowStatus.dropped]: 'Брошено',
}

export const STATUS_OPTIONS = Object.values(UserShowStatus).map((value) => ({
  value,
  label: STATUS_LABELS[value] ?? value,
}))
