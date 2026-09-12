# КосмоХакатон NN — backend

Сервис проектирования устойчивой спутниковой группировки. Один процесс, три слоя: **api → service → repository**. Геометрия считается эталонным `python/geometry.py` без правок.

Целевой ориентир ТЗ: доступность сквозного пути **client → спутники → gateway ≥ 90%** для каждого наземного пункта. Наземные пункты не ретранслируют трафик.

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

`POST /api/projects` принимает `cosmo-A-1.0` или выгрузку `cosmo-A-result-1.0`. Идентификатор проекта всегда новый UUID, `meta.id` не используется.

`PATCH` меняет `launch_stage`, `planes[].raan_deg` / `phase_deg`, `failures`, `gateway_outages`. Некорректный вход возвращает 400 с указанием поля.

`POST /api/compare` сравнивает два прогона (`run_a` + `run_b`) или список `run_ids`. В ответе: `config_diff`, метрики по пунктам, mean hops, рекомендация с преимуществами, условиями цели 90% и ограничениями.

Маршруты: BFS min hops. Причины разрыва: `no_visible_sat`, `no_gateway_contact`, `gateway_outage`, `isl_partition`. Выгрузка: `schema_version=cosmo-A-result-1.0`, `effective_scenario`, `routes[].{t_s,client_id,path,reason,hops}`, метрики с интервалами `gaps`. `POST /api/runs/{id}/what-if` сравнивает отказ с исходным прогоном, но не рекомендует run_b как рабочий конфиг.

## Расчёт

Сетка `range(0, horizon_s, step_s)` — правый конец горизонта не входит. Активны КА с `launch_batch <= launch_stage` без отказа. Контакты считает `geometry.py`. Показатели: доля видимости, доля пути, max перерыв (начало и конец периода — отдельные интервалы, без «склейки» через полночь), среднее число hops.
