export interface LegalOperator {
  name: string
  kind: 'Физическое лицо' | 'Индивидуальный предприниматель' | 'Юридическое лицо'
  inn: string
  email: string
  /** Confirmed production hosting, countries and third-party data recipients. */
  infrastructure: string
  /** Actual retention and manual deletion procedure, including backups/logs. */
  retention: string
  updatedAt: string
}

// Fill with the owner's confirmed details before publishing. Do not expose
// invented requisites or an incomplete privacy policy on the production site.
export const legalOperator: LegalOperator | null = {
  name: 'Байдуров Алексей Владимирович',
  kind: 'Физическое лицо',
  inn: '660685531389',
  email: 'deface06@yandex.ru',
  infrastructure:
    'Прод-версия сервиса shows.deface.dev размещена на инфраструктуре хостинг-провайдера ' +
    'H3llo Cloud (https://h3llo.cloud/). Передача персональных данных иным третьим лицам, ' +
    'помимо указанных в разделах политики (в частности, TMDB и Telegram), не осуществляется.',
  retention:
    'Данные аккаунта и учёта просмотра хранятся до удаления аккаунта по обращению ' +
    'пользователя на указанный контактный адрес; резервные копии хранятся до 30 дней ' +
    'и затем удаляются в ходе их обычной ротации.',
  updatedAt: '21 сентября 2026',
}
