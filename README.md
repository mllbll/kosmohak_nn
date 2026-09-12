# КосмоХакатон NN — backend

Сервис проектирования устойчивой спутниковой группировки. Один процесс, три слоя: **api → service → repository**. Геометрия считается эталонным `python/geometry.py` без правок.

Целевой ориентир ТЗ: доступность сквозного пути **client → спутники → gateway ≥ 90%** для каждого наземного пункта. Наземные пункты не ретранслируют трафик.

Внутреннее устройство бэкенда (слои, алгоритмы, расширение): [docs/backend.md](docs/backend.md). Контракт HTTP для фронтенда: [docs/frontend.md](docs/frontend.md). Экраны, localStorage, what-if / alternatives / explore: [docs/frontend-ui.md](docs/frontend-ui.md). TypeScript-клиент (скопировать в UI): [docs/frontend-client.ts](docs/frontend-client.ts). Деплой Docker: [docs/docker.md](docs/docker.md).

## Слои

```
cmd/main.go
internal/
  api/            HTTP (chi), один хендлер — один файл
  app/            сборка приложения и DI
  client/         вызов geometry.py
  model/          доменные сущности и Request/Response
  repository/     in-memory хранение проектов и прогонов
  service/        бизнес-логика: патч, маршрут, метрики, compare
```

Хранение в памяти процесса. Kafka/Redis/Postgres нет.

## Зависимости и запуск

Нужны Go 1.23+ и Python 3.10+ с NumPy.

```bash
python3 -m pip install numpy
make tidy
make run
```

Сервис слушает `:8080` (`HTTP_ADDR` меняет адрес). Проверка эталона:

```bash
python3 python/runner.py data/01_full_constellation.json 0
make test
```

Переменные: `HTTP_ADDR`, `PYTHON_BIN`, `RUNNER_SCRIPT`, `CORS_ORIGINS`.

## Docker

Полная инструкция: **[docs/docker.md](docs/docker.md)**.

На любой машине с Docker:

```bash
docker compose up --build
```

Образ сам собирает Go-бинарник и кладёт рядом Python + NumPy + `geometry.py`. API: `http://localhost:8080`. Остановка: `Ctrl+C`, затем `docker compose down`.

Только образ, без compose:

```bash
docker build -t kosmohak-nn .
docker run --rm -p 8080:8080 kosmohak-nn
```

## Демонстрационный сценарий (ТЗ)

1. Загрузить полную группировку и посчитать сутки:
   ```bash
   curl -s -X POST http://localhost:8080/api/projects \
     -H 'Content-Type: application/json' \
     --data-binary @data/01_full_constellation.json
   ```
2. `POST /api/projects/{id}/runs` — 720 шагов (0…86280, шаг 120 с).
3. `GET /api/runs/{id}/snapshot?t_s=0&client_id=C65` — сеть, маршрут BFS (min hops), видимые КА, дельта относительно предыдущего шага.
4. Сравнить полную группировку с первой очередью: `PATCH` `launch_stage=1`, второй прогон, `POST /api/compare`.
5. Отказ аппарата: `POST /api/runs/{id}/what-if` с `satellite_id` с маршрута.
6. Выгрузка `GET /api/runs/{id}/export` (`cosmo-A-result-1.0`). Этот JSON можно снова отдать в `POST /api/projects` — берётся `effective_scenario`.
7. Сброс правок: `POST /api/projects/{id}/reset`. Копия варианта: `POST /api/projects/{id}/copy`.

Ожидаемые порядки на суточных фикстурах:

| Файл | mean path_ratio | цель 90% |
|---|---|---|
| `01_full_constellation.json` | ~0.98 | все пункты |
| `02_first_launch.json` | ~0.19 | никто |
| `03_satellite_outages.json` | ~0.81 | никто |
| `04_link_range.json` | ~0.68 при видимости ~98–100% | ISL partition |

## API

Как устроен сам сервис изнутри: **[docs/backend.md](docs/backend.md)**. Полный контракт для фронтенда (схемы, статусы, примеры, потоки UI, фикстуры, чего нет): **[docs/frontend.md](docs/frontend.md)**. Как собрать экраны без догадок: **[docs/frontend-ui.md](docs/frontend-ui.md)**.

| Метод | Путь |
|---|---|
| GET | `/health` |
| POST | `/api/projects` |
| GET | `/api/projects/{id}` |
| PATCH | `/api/projects/{id}` |
| POST | `/api/projects/{id}/reset` |
| POST | `/api/projects/{id}/copy` |
| POST | `/api/projects/{id}/runs` |
| GET | `/api/runs/{id}` |
| GET | `/api/runs/{id}/metrics` |
| GET | `/api/runs/{id}/snapshot?t_s=&client_id=` |
| GET | `/api/runs/{id}/export` |
| POST | `/api/runs/{id}/what-if` |
| POST | `/api/compare` |

CORS `*`, авторизации нет. `POST /api/projects` принимает `cosmo-A-1.0` или выгрузку `cosmo-A-result-1.0`; `project.id` всегда новый UUID. Create/copy/run/what-if отвечают **201**. Ошибки: `{ "error": "..." }` и 400/404/422/500.

## Расчёт

Сетка `range(0, horizon_s, step_s)` — правый конец горизонта не входит. Активны КА с `launch_batch <= launch_stage` без отказа. Контакты считает `geometry.py`. Показатели: доля видимости, доля пути, max перерыв (начало и конец периода — отдельные интервалы, без «склейки» через полночь), среднее число hops.
