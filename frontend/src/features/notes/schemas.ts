import { z } from 'zod'

// Note scope mirrors the backend enum (show | season | episode). Season/episode
// numbers are required only for the scopes that reference them.
export const noteSchema = z
  .object({
    scope: z.enum(['show', 'season', 'episode']),
    season_number: z.number().int().positive().nullable(),
    episode_number: z.number().int().positive().nullable(),
    body: z.string().trim().min(1, 'Заметка не может быть пустой').max(5000, 'Слишком длинная заметка'),
  })
  .refine((v) => v.scope === 'show' || v.season_number != null, {
    message: 'Укажите номер сезона',
    path: ['season_number'],
  })
  .refine((v) => v.scope !== 'episode' || v.episode_number != null, {
    message: 'Укажите номер эпизода',
    path: ['episode_number'],
  })
export type NoteInput = z.infer<typeof noteSchema>

export const noteBodySchema = z.object({
  body: z.string().trim().min(1, 'Заметка не может быть пустой').max(5000, 'Слишком длинная заметка'),
})
export type NoteBodyInput = z.infer<typeof noteBodySchema>
