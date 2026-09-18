# defShows Frontend

Веб-клиент defShows поверх бэкенд-API: auth (email + OAuth), поиск/трекинг сериалов,
эпизоды с прогрессом, рейтинги/озвучки/ссылки, приватные заметки и recap, уведомления
+ Telegram, админка справочника озвучек.

## Стек
- **React 19 + Vite + TypeScript (strict)**
- **Mantine 9** — UI + `@mantine/notifications`, `@mantine/dates`
- **TanStack Query** — server-state; **zustand** — auth-state
- **orval** — кодоген TanStack Query-хуков/типов из OpenAPI-контрактов бэка
- **react-hook-form + zod** — формы/валидация
- **Vitest + React Testing Library + MSW** — тесты (мок сети)
- **React Router 7**

Структура — Feature-Sliced Design lite (`app` / `pages` / `features` / `entities` / `shared`).

## Скрипты
```bash
npm run dev        # dev-сервер (порт 8080)
npm run build      # tsc -b && vite build
npm run test       # vitest run
npm run test:watch # vitest в watch-режиме
npm run lint       # eslint
npm run api:gen    # перегенерация API-клиента из ../backend/api/openapi/*.yaml
```

## API-клиент (orval)
Хуки и типы генерируются из контрактов бэка в `src/shared/api/<name>/` (auth, shows,
tracking, notifications, notes, admin). **Сгенерённый код не редактируем руками** —
меняем контракт в бэке и гоняем `npm run api:gen`. Мутации оборачиваем через
`useMutation` поверх raw-функций (в конфиге включён только `useQuery`).

## Runtime-конфигурация
Базовый URL API берётся в рантайме из `window.__ENV.API_BASE_URL`
(`src/shared/lib/env.ts`), с фолбэком на `VITE_API_BASE_URL` и `http://localhost:8080`.
`index.html` подключает `/env.js`:
- **dev** — `public/env.js` с плейсхолдером `${API_BASE_URL}` (игнорируется, идёт фолбэк);
- **prod** — `docker-entrypoint.sh` рендерит `env.js` из `public/env.template.js` через
  `envsubst` при старте контейнера.

## Docker
`Dockerfile` — multi-stage (node build → nginx). `env.js` подставляется entrypoint-хуком
из переменной `API_BASE_URL`. Сервис `frontend` заведён в
`../backend/deploy/docker-compose.yml` (profile `app`, порт `FRONTEND_PORT`, по умолчанию 3000).
