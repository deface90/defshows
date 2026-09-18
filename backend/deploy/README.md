# deploy — стенды docker-compose

Два стенда:

- **`docker-compose.yml`** — DEV. Порты хранилищ (Postgres/MinIO) открыты наружу
  для инструментов и отладки. Все бэкенд-сервисы собираются локально из **одного**
  образа (`deploy/Dockerfile`) и отличаются только `command`.
- **`docker-compose.prod.yml`** — PROD. Готовые образы из GHCR (собирает CI),
  порты Postgres/MinIO наружу **не** публикуются, снаружи доступны только SPA
  (`3000`) и web-API (`8088`).

## Один образ бэка

`deploy/Dockerfile` собирает все `cmd/*` (`auth`, `web`, `notifier`, `worker`) в
один образ. Нужный сервис выбирается командой запуска: `command: ["web"]` и т.п.
(по умолчанию `web`).

## DEV — быстрый старт

```bash
cp .env.example .env          # заполнить ключи (TMDB/OMDb/Telegram/OAuth/JWT)

# только инфра (postgres + minio + бакет)
docker compose up -d
# или из backend/: make up

make -C .. migrate-up          # применить миграции

# вместе с сервисами
docker compose --profile app up -d --build
```

Порты (dev): Postgres `5432`, MinIO API `9000`, MinIO Console `9001`
(`minioadmin`/`minioadmin`), web `8080`, auth `8081`, frontend `3000`.

## PROD — запуск

```bash
cp .env.example .env          # секреты + BACKEND_IMAGE/FRONTEND_IMAGE + FRONTEND_API_BASE_URL

docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

Публикуются только `frontend` (`FRONTEND_PORT`, деф. `3000`) и `web`
(`WEB_PORT`, деф. `8088`). Postgres/MinIO/notifier/worker — только внутри сети
compose. Порты и теги образов переопределяются через `.env` (`WEB_PORT`,
`FRONTEND_PORT`, `BACKEND_IMAGE`, `FRONTEND_IMAGE`).

Отдельный `auth`-сервис в prod по умолчанию **не** запускается: все `/auth/*`
роуты (login/register/refresh/oauth/callback/exchange) обслуживает `web` — туда
и ходят SPA и OAuth-callback (`OAUTH_REDIRECT_BASE_URL`). Standalone `auth`
нужен только для разнесённого деплоя (свой хост + свой публичный URL) — в
`docker-compose.prod.yml` он оставлен закомментированным.

`FRONTEND_API_BASE_URL` — браузерный URL web-API (напр. `https://api.example.com`
или `http://<host>:8088`); зашивается в `env.js` при старте фронта.
`CORS_ALLOWED_ORIGINS` должен включать origin фронта.

## CI — сборка образов

`.github/workflows/build-images.yml` собирает и пушит в GHCR два образа —
`ghcr.io/deface90/defshows-backend` и `-frontend`:

- push в `main` → тег `main` + `latest` + `sha-<...>`;
- тег `vX.Y.Z` → `X.Y.Z`, `X.Y`, `latest`;
- pull request → только сборка (без пуша), проверка Dockerfile'ов.

## Заметки

- `minio-setup` — одноразовый контейнер: создаёт бакет `${MINIO_BUCKET}` и выходит.
- Данные Postgres/MinIO — в именованных volume'ах (`pgdata`, `miniodata`).
