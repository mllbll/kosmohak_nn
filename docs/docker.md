# Деплой через Docker

Сервис поднимается одним образом: Go API + Python 3.12 + NumPy + эталонный `python/geometry.py`. На хосте **не нужны** Go и Python — достаточно Docker.

Контракт HTTP: [frontend.md](frontend.md). Код приложения для образа не меняется.

## Что нужно на машине

- Docker Engine 24+ (Linux) **или** Docker Desktop (macOS / Windows) **или** Colima
- Docker Compose v2 (`docker compose`, не `docker-compose`)
- Порт **8080** свободен

Проверка:

```bash
docker version
docker compose version
```

## Быстрый старт

Из корня репозитория:

```bash
docker compose up --build
```

Эквивалент через Makefile: `make docker-run`.

После логов `http server listening on :8080`:

```bash
curl -s http://localhost:8080/health
# {"status":"ok"}
```

Остановка: `Ctrl+C`, затем `docker compose down`.

Фоном:

```bash
docker compose up --build -d
docker compose logs -f api
docker compose down
```

## Только образ, без Compose

```bash
docker build -t kosmohak-nn .
docker run --rm -p 8080:8080 --name kosmohak-nn kosmohak-nn
```

Сборка образа: `make docker-build`.

## Что внутри образа

Многостадийная сборка (`Dockerfile`):

| Стадия | База | Результат |
|---|---|---|
| `build` | `golang:1.23-bookworm` | статический бинарник `/out/api` (`CGO_ENABLED=0`) |
| runtime | `python:3.12-slim-bookworm` | `/app/api` + NumPy + `python/` + `data/` |

Процесс в контейнере:

- рабочая директория `/app`
- пользователь `nobody` (не root)
- вход: `/app/api`
- Python вызывается как `python3 python/runner.py` (тот же `RUNNER_SCRIPT`, что и локально)
- фикстуры лежат в `/app/data` — удобно для `docker compose exec`

Healthcheck раз в 15 с бьёт в `http://127.0.0.1:8080/health`.

## Порты и URL

| Где | Адрес |
|---|---|
| с хоста / фронт на той же машине | `http://localhost:8080` |
| из другого контейнера в той же compose-сети | `http://api:8080` |
| внутри контейнера | `http://127.0.0.1:8080` |

Проброс задаётся в `docker-compose.yml`: `"8080:8080"`. Другой порт на хосте:

```bash
docker run --rm -p 3000:8080 kosmohak-nn
# API: http://localhost:3000
```

Или в compose:

```yaml
ports:
  - "3000:8080"
```

Внутри контейнера сервис всё равно слушает `:8080` (`HTTP_ADDR`).

## Переменные окружения

Совпадают с локальным запуском (`internal/config/config.go`).

| Переменная | По умолчанию в образе | Смысл |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес listen; в Docker оставляйте `:8080` |
| `PYTHON_BIN` | `python3` | интерпретатор геометрии |
| `RUNNER_SCRIPT` | `python/runner.py` | путь **относительно `/app`** |
| `CORS_ORIGINS` | `*` | origins через запятую |

Пример — только фронт с `http://localhost:5173`:

```bash
docker run --rm -p 8080:8080 \
  -e CORS_ORIGINS=http://localhost:5173 \
  kosmohak-nn
```

В compose те же ключи в `environment:`. Не меняйте `PYTHON_BIN` и `RUNNER_SCRIPT`, если не кладёте свой раннер.

Авторизации нет. `CORS_ORIGINS=*` нельзя сочетать с `credentials: 'include'` на фронте (`AllowCredentials: false`).

## Данные и рестарт

Хранение **только в памяти процесса**. `docker compose restart`, `down`, пересборка образа и падение контейнера **стирают** проекты и прогоны.

Списка проектов в API нет — фронт сам хранит `id`. После деплоя заново создайте проект (`POST /api/projects`) или загрузите export.

Тома для API не нужны: БД нет. Фикстуры уже в образе.

## Таймауты

HTTP-таймаут сервера — **5 минут**. Суточный прогон (`POST /api/projects/{id}/runs`, 720 шагов) и what-if могут идти десятки секунд. На фронте и в `curl` ставьте длинный timeout (`curl --max-time 300`).

Прокси (nginx, Caddy, облачный LB) тоже должен держать соединение не меньше 5 минут, иначе клиент получит обрыв, а не JSON ошибки.

## Другая архитектура CPU

`docker compose build` / `docker build` собирают образ **под машину, где идёт build** (Apple Silicon → `linux/arm64`, обычный PC → `linux/amd64`). Этого достаточно, чтобы запустить у себя.

Собрать под Linux-сервер с Mac:

```bash
docker build --platform linux/amd64 -t kosmohak-nn:amd64 .
```

Запуск на сервере:

```bash
docker load < kosmohak-nn.tar   # если передали tar
docker run --rm -p 8080:8080 kosmohak-nn:amd64
```

## Проверка после деплоя

```bash
curl -sS http://localhost:8080/health

curl -sS -X POST http://localhost:8080/api/projects \
  -H 'Content-Type: application/json' \
  --data-binary @data/01_full_constellation.json
```

Ожидается **201** и JSON проекта с новым `id` (не `meta.id` из файла).

Геометрия внутри контейнера:

```bash
docker compose exec api python3 python/runner.py data/01_full_constellation.json 0
```

Должен печататься JSON снимка (`t_s`, `satellites`, `edges`).

## Типичные сбои

| Симптом | Что проверить |
|---|---|
| `Cannot connect to the Docker daemon` | запущены Docker Desktop / Colima (`colima start`) |
| порт занят | `lsof -i :8080` или смените проброс `-p 3000:8080` |
| `curl: (7) Failed to connect` сразу после `up` | подождите 1–2 с, пока процесс слушает; статус: `docker compose ps` (должен быть `healthy`) |
| 422 на create/run | Python/NumPy в образе сломаны или `RUNNER_SCRIPT` указывает не туда; смотрите `docker compose logs api` |
| CORS во фронте | origin в `CORS_ORIGINS`; при `*` не используйте cookie credentials |
| пустые проекты после рестарта | ожидаемо: память процесса, не диск |

Логи:

```bash
docker compose logs -f api
```

## Файлы в репозитории

| Файл | Роль |
|---|---|
| `Dockerfile` | сборка образа |
| `docker-compose.yml` | один сервис `api`, порт 8080, restart `unless-stopped` |
| `.dockerignore` | в контекст не попадают git, тесты, docs |
| `Makefile` | `docker-build`, `docker-run` |

Образ **не содержит** исходники тестов и `docs/` — в runtime только бинарник, Python и фикстуры.
