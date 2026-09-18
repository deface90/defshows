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
