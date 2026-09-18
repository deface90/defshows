import { z } from 'zod'

export const dubbingStudioSchema = z.object({
  name: z.string().trim().min(1, 'Введите название'),
  site_url: z.union([z.string().url('Некорректный URL'), z.literal('')]).optional(),
  active: z.boolean(),
})
export type DubbingStudioInput = z.infer<typeof dubbingStudioSchema>
