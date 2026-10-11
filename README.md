# filebox

Мини-S3: HTTP-сервис для загрузки и скачивания файлов. Пет-проект для практики DevOps: контейнеризация, CI/CD, оркестрация, мониторинг.

> **Цель проекта** — не продукт для отработки полного DevOps-цикла: от кода до продакшн-подобного деплоя.

---

## Что умеет

- **Загрузка файлов** — `POST /files`
- **Скачивание файлов** — `GET /files/{id}`
- **Проверка живости** — `GET /healthz`
- **Метаданные в Postgres** — id, имя, размер, тип, дата
- **Миграции БД** — через goose
- **Reverse proxy** — Caddy с автоматическим HTTPS
- **CI/CD** — GitHub Actions: сборка, тесты, push в GHCR, деплой

---

## Стек

| Слой | Технология |
|------|-----------|
| Язык | Go 1.27 |
| HTTP | стандартная библиотека `net/http` |
| БД | PostgreSQL 16 |
| Драйвер БД | `pgx/v5` + `pgxpool` |
| Миграции | `goose` |
| Контейнеризация | Docker, multi-stage build |
| Оркестрация (локально) | Docker Compose |
| Reverse proxy | Caddy 2 |
| Registry | GitHub Container Registry (GHCR) |
| CI/CD | GitHub Actions |
| Хранилище файлов | локальный volume (план: MinIO) |

---

## Архитектура

```
                        ┌──────────────┐
                        │   интернет   │
                        └──────┬───────┘
                               │
                        ┌──────▼───────┐
                        │    Caddy     │  :80, :443
                        │ reverse proxy│  HTTPS, redirect
                        └──────┬───────┘
                               │ app:8000
                        ┌──────▼───────┐
                        │     app      │  Go, :8000
                        │  HTTP server │  upload/download
                        └──────┬───────┘
                               │ db:5432
                        ┌──────▼───────┐
                        │  PostgreSQL  │  :5432
                        │  метаданные  │  volume: pgdata
                        └──────────────┘

        volume: ./data  ←  сюда сохраняются файлы
```

Все сервисы — в одной Docker-сети. Наружу открыты только порты Caddy.

---

## API

### `POST /files`

Загрузка файла. Multipart form, поле `file`.

```bash
curl -F "file=@test.txt" https://localhost/files
```

Ответ:
```json
{"id":"550e8400-e29b-41d4-a716-446655440000","url":"/files/550e8400-e29b-41d4-a716-446655440000"}
```

### `GET /files/{id}`

Скачивание файла по id. Возвращает файл с оригинальным именем.

```bash
curl https://localhost/files/550e8400-e29b-41d4-a716-446655440000
```

### `GET /healthz`

Проверка, что сервис жив. Используется в Docker healthcheck и CI.

```bash
curl https://localhost/healthz
# → {"status":"ok"}
```

---

## Быстрый старт

### Требования

- Docker Desktop
- Go 1.27 (для локальной разработки без Docker)

### Запуск

```bash
git clone https://github.com/dashatuchka/filebox.git
cd filebox

# создать .env из шаблона
cp .env.example .env
# отредактировать .env: придумать пароль для Postgres

# поднять всё
docker compose up -d
```

Сервис будет доступен на `https://localhost` (self-signed сертификат).

### Проверка

```bash
curl -k https://localhost/healthz
# → {"status":"ok"}

echo "hello filebox" > test.txt
curl -k -F "file=@test.txt" https://localhost/files
# → {"id":"...","url":"/files/..."}
```

---

## Переменные окружения

| Переменная | Описание | Пример |
|------------|----------|--------|
| `POSTGRES_USER` | пользователь Postgres | `filebox` |
| `POSTGRES_PASSWORD` | пароль Postgres | `change_me` |
| `POSTGRES_DB` | имя базы | `filebox` |
| `DATABASE_URL` | строка подключения | `postgres://filebox:change_me@db:5432/filebox?sslmode=disable` |

Шаблон — в `.env.example`. Реальный `.env` **не коммитится**.

---

## Структура проекта

```
filebox/
├── .github/
│   └── workflows/
│       └── ci.yml              # CI/CD pipeline
├── migrations/
│   ├── 001_create_files.sql
│   └── 002_add_mime_type.sql
├── app/
│   └── (план: разнести по пакетам)
├── main.go                     # HTTP-сервер, хендлеры
├── db.go                       # подключение к Postgres, goose
├── main_test.go                # тесты
├── Dockerfile                  # multi-stage build
├── docker-compose.yml          # app + db + caddy
├── Caddyfile                   # конфиг reverse proxy
├── .dockerignore
├── .env.example
└── README.md
```

---

## CI/CD

### CI (`.github/workflows/ci.yml`)

Триггер: push в `main`, pull request.

1. **Checkout** — код из репозитория
2. **Set up Go** — установка Go 1.27
3. **Build** — `go build ./...`
4. **Test** — `go test ./...`
5. **Log in to GHCR** — авторизация в реестре
6. **Build and push Docker image** — сборка и push в GHCR с двумя тегами (`latest` и `${{ github.sha }}`)

### CD (self-hosted runner)

Job `deploy-local` выполняется на self-hosted runner (macOS):

1. `git fetch origin && git reset --hard origin/main`
2. `docker compose pull` — скачать свежий образ из GHCR
3. `docker compose up -d` — перезапустить
4. `docker compose ps` — проверить статус

**План:** переехать на VPS + deploy через SSH + reverse proxy с настоящим доменом.

---

## Что демонстрирует

Проект показывает полный DevOps-цикл:

- **Go** — HTTP-сервер, работа с БД, миграции, тесты
- **Docker** — multi-stage build, оптимизация размера образа
- **Docker Compose** — multi-service, volumes, healthchecks, networks
- **PostgreSQL** — схема, индексы, миграции через goose
- **Reverse proxy** — Caddy, автоматический HTTPS, redirect
- **CI/CD** — GitHub Actions, self-hosted runner, GHCR
- **Инфраструктура как код** — все сервисы описаны в YAML
- **Работа с секретами** — `.env`, `.env.example`, GitHub Secrets
- **Отладка** — `docker compose logs`, `curl -v`, `psql`, сеть Docker

---

## Roadmap

- [x] HTTP-сервер на Go
- [x] Dockerfile (multi-stage)
- [x] Docker Compose + Postgres
- [x] Подключение БД к коду
- [x] CI (GitHub Actions)
- [x] CD (self-hosted runner)
- [x] Push образа в GHCR
- [x] Reverse proxy (Caddy)
- [x] Миграции (goose)
- [ ] Мониторинг (Prometheus + Grafana)
- [ ] Логи (Loki)
- [ ] Хранилище файлов в MinIO (S3-совместимое)
- [ ] Graceful shutdown
- [ ] `/readyz` с проверкой БД
- [ ] Security scanning (Trivy, gosec)
- [ ] Деплой на VPS
- [ ] Terraform (инфраструктура как код)
- [ ] Kubernetes (k3d локально → k3s на VPS)

---

## Лицензия

MIT