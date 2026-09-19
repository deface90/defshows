# defShows — backend

Go (echo) распределённый монолит. Общий обзор проекта и quickstart — в
[корневом README](../README.md). 

## Структура

```
cmd/                 бинари: auth, web, notifier, worker
internal/
  service/           entity, repository, usecase, workers, consumer
  gateways/          входящие (http) и исходящие (провайдеры) адаптеры
pkg/                 переиспользуемое: config, db, auth, server (gen), clients (gen), ...
api/openapi/         контракты сервиса → codegen в pkg/server
deps/<svc>/openapi/  контракты сторонних API → codegen в pkg/clients/<svc>
migrations/migrate/  goose SQL-миграции
deploy/              docker-compose, конфиги хранилищ, .env.example
```

## Команды

```bash
make help          # список задач
make build         # сборка
make test          # тесты
make migrate-up    # применить миграции (DB_DSN из env)
make generate      # кодоген
make up / make down
```


## Изображения TMDB

Браузер получает внешние постеры через публичный `GET /images/tmdb/:size/:filename`,
например `/images/tmdb/w500/abc123.jpg`. Сервис `web` скачивает их с фиксированного
`https://image.tmdb.org/t/p/` через HTTP-клиент с `PROXY_URL`. MinIO и добавление
сериала в личный список для этого не нужны. При пустом `PROXY_URL` сервер скачивает напрямую.

Фронтенд заменяет TMDB URL при отображении постеров в поиске, подборе, карточке
сериала и списках. Адрес API берётся из существующего `API_BASE_URL`.
Сохранённые в S3 изображения продолжают обслуживаться через `/images/*`.

Ответы кешируются браузером на сутки; поддерживаются `ETag` / `Last-Modified`
и условные запросы. Отдельного серверного кеша для этого маршрута нет.
Прокси принимает только разрешённые размеры и имена JPG/PNG/WebP, не следует
редиректам, ограничивает время запроса и размер изображения (10 MiB).
Ошибки провайдера возвращаются как 502, отсутствующие изображения — как 404.

Формат адресов: [TMDB Image Basics](https://developer.themoviedb.org/docs/image-basics).
