# defShows

Личный трекер сериалов: каталог (зеркало TMDB), трекинг просмотренных эпизодов/сезонов,
отслеживание выхода новых серий, автоматические ссылки на IMDb и Wikipedia, агрегатор рейтингов,
приватные заметки и публичные сгенерированные описания (recap), уведомления в Telegram.

Monorepo:

| Папка       | Что это                                                                 |
|-------------|-------------------------------------------------------------------------|
| `backend/`  | Go (echo) — распределённый монолит: `auth`, `web`, `notifier`, `worker` |
| `frontend/` | React 18 + Vite + TS (Mantine, TanStack Query)                          |

## Архитектура

Распределённый монолит: несколько бинарей (`cmd/*`) шарят одну Postgres БД, без шины
сообщений. Хранилище файлов — MinIO (S3). Миграции — goose, ORM — gorm. Контракты —
OpenAPI + кодоген (oapi-codegen на бэке, orval на фронте).

## Быстрый старт (dev)

```bash
# поднять стенд (postgres + minio + сервисы)
make -C backend up
# применить миграции
make -C backend migrate-up
# фронт в dev-режиме
cd frontend && npm install && npm run dev
```

Переменные окружения — см. `backend/deploy/.env.example`.
