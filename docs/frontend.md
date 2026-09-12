# Контракт API для фронтенда

Источник истины — текущий Go-код (`internal/app/app.go`, `internal/api/**`, `internal/model/**`, `internal/service/**`). Эндпоинтов, которых нет в таблице ниже, **нет**. Не вызывайте `python/geometry.py` из браузера: геометрию считает сервер.

База: `http://localhost:8080` (переменная `HTTP_ADDR`, по умолчанию `:8080`).

## 1. Запуск

Нужны Go 1.23+ и Python 3.10+ с NumPy.

```bash
python3 -m pip install numpy
make tidy
make run
```

Переменные:

| Переменная | По умолчанию | Смысл |
|---|---|---|
| `HTTP_ADDR` | `:8080` | адрес HTTP |
| `CORS_ORIGINS` | `*` | список origin через запятую |
| `PYTHON_BIN` | `python3` | интерпретатор геометрии |
| `RUNNER_SCRIPT` | `python/runner.py` | раннер снимков |

Хранение **в памяти процесса**. Рестарт сервера стирает проекты и прогоны. Списка проектов/прогонов в API нет — фронт сам держит `id` после создания.

Таймаут запроса: **5 минут** (chi `middleware.Timeout` v5.3.2). Суточный прогон (720 шагов) и what-if могут идти десятки секунд — ставьте длинный timeout на fetch. Контекст отменяется; python-процесс убивается. Если хендлер после этого успел записать JSON (часто **422** `geometry calculation failed`), **504 не придёт**. 504 (`WriteHeader` без тела) бывает только если хендлер вышел, не вызвав `WriteHeader`. Обрабатывайте и 422, и 504.

## 2. Общие правила HTTP

- Авторизации **нет**. Заголовки `Authorization` CORS пропускает, сервер их не читает.
- CORS: origins из `CORS_ORIGINS` (по умолчанию `*`), методы `GET POST PUT PATCH DELETE OPTIONS`, заголовки `Accept, Authorization, Content-Type`, **`AllowCredentials: false`**. Не используйте `credentials: 'include'` с дефолтным `*`.
- CORS разрешает `PUT`/`DELETE`, но таких маршрутов нет → **405** от chi, тело не `{error}`.
- Тела запросов — JSON. Лишние поля сервер **игнорирует** (`DisallowUnknownFields` нет).
- Ответы API: `Content-Type: application/json; charset=utf-8`. Неизвестный URL — chi **404** `text/plain` (`404 page not found`), не `{error}`. Неверный метод — **405**, тоже не `{error}`.
- Успех без обёртки `{data: ...}`: сразу сущность или массив.
- `GET /health` не проверяет Python/NumPy.

### Коды ошибок API

Тело штатных ошибок API:

```json
{"error":"invalid argument: launch_stage must be 1, 2 or 3"}
```

Исключения: chi 404/405 и редкий 504 — не этот JSON.

| HTTP | Когда |
|---|---|
| 400 | невалидный JSON, query, сценарий, `t_s`, `client_id`, compare без двух id |
| 404 | неизвестный `project` / `run` (`project not found` / `run not found`) |
| 422 | сбой расчёта geometry.py / несовпадение сетки (`geometry calculation failed: ...`) |
| 500 | прочее |
| 504 | только если хендлер не успел записать ответ после дедлайна 5 мин; иначе чаще 422 |

Часть сообщений 400 содержит путь поля (`planes[0].id is empty`, `unknown plane id "P9"`), часть — общая (`invalid orbit`, `invalid time grid`). Не завязывайтесь на парсинг текста сверх показа пользователю.

Python `ValueError`/`TypeError`/`KeyError`/`JSONDecodeError` приходят как **400**, остальные падения python — **422**.

## 3. omitempty, false/0, [] vs null

| Правило | Поведение |
|---|---|
| `meets_target`, `meets_target_a/b`, `changed`, `previous_still_valid`, `active` | **всегда в JSON**, в том числе `false` |
| `max_gap_s`, `mean_hops`, `path_ratio`, … | **всегда**, в том числе `0` |
| `reason`, `hops`, `alternatives`, `min_hops` | `omitempty`: при успехе `reason` нет; при разрыве нет `hops` (это 0) |
| `previous_path` / `current_path` | `omitempty`: пустой путь **отсутствует**, не `[]` |
| `config_diff`, `run_a_id`, `run_b_id` | только если сравнивают ровно 2 прогона; пустой diff может **пропасть** |
| `recommendation.run_id` | нет при ничьей |
| `failures` / `gateway_outages` / `gaps` / `path` в ответах API | после clone/export — **`[]`, не `null`** |
| `visible_satellites` | всегда массив, может быть `[]` |

На входе: отсутствующий массив в сценарии после create станет `[]`. Для PATCH см. ниже: `null` ≠ `[]`.

## 4. Глоссарий

| Термин | JSON | Смысл |
|---|---|---|
| Сценарий | `cosmo-A-1.0` | входная группировка: орбита, плоскости, КА, пункты |
| Проект | `Project` | `id` (UUID сервера) + `base` (как загрузили) + `effective` (после PATCH) |
| `meta.id` | строка фикстуры (`01_full_constellation`) | **не** id проекта; create всегда новый UUID |
| Прогон / run | `Run` | расчёт по **effective** на сетке `range(0, horizon_s, step_s)` (правый конец **не** входит) |
| Снимок | `snapshot` | позиции КА и рёбра в момент `t_s` (выход geometry.py) |
| Маршрут | `route` / `path` | BFS min hops: `client → спутники → gateway`. Земля **не** ретранслирует |
| hops | `hops` | `len(path) - 1`; на валидном пути ≥ 2 |
| alternatives | `alternatives` | другие пути той же длины (до 3) и +1 hop (до 2) |
| network_delta | `network_delta` | сравнение с **предыдущим шагом сетки** строго раньше `t_s` |
| visible_satellites | `visible_satellites` | активные КА с возвышением ≥ `min_elevation_deg` у выбранного клиента |
| path_ratio | `path_ratio` | доля шагов сетки с непустым путём |
| visibility_ratio | `visibility_ratio` | доля шагов, где клиент видит хотя бы один активный КА |
| max_gap_s | `max_gap_s` | самый длинный непрерывный разрыв пути, в секундах (`шаги * step_s`); полночь **не** склеивается |
| gaps | `gaps[]` | интервалы разрыва `[start_s, end_s)` по сетке; `end_s` = время последнего плохого шага + `step_s` |
| meets_target | `meets_target` | `path_ratio >= target_availability` (в фикстурах 0.9) |
| compare | `/api/compare` | сравнение 2+ уже посчитанных прогонов |
| what-if | `/what-if` | новый проект + новый прогон с отказом; **не** правка исходного |

Причины разрыва (`reason`):

| Значение | Когда |
|---|---|
| `no_visible_sat` | у клиента нет аплинка на активный КА |
| `no_gateway_contact` | ни один живой шлюз не имеет даунлинка |
| `gateway_outage` | все шлюзы в outage на этом `t` |
| `isl_partition` | аплинк и шлюз есть, межспутниковая сеть не связывает |

На успешном шаге поле `reason` отсутствует.

Активный КА: `launch_batch <= launch_stage` и не попадает в `failures` на времени `t` (`start_s <= t < end_s`).

## 5. Эндпоинты (все)

| Метод | Путь | Успех | Тело запроса |
|---|---|---|---|
| `GET` | `/health` | 200 | нет |
| `POST` | `/api/projects` | **201** | сценарий **или** выгрузка result |
| `GET` | `/api/projects/{id}` | 200 | нет |
| `PATCH` | `/api/projects/{id}` | 200 | `Patch` |
| `POST` | `/api/projects/{id}/reset` | 200 | нет (игнорируется) |
| `POST` | `/api/projects/{id}/copy` | **201** | нет |
| `POST` | `/api/projects/{id}/runs` | **201** | нет |
| `GET` | `/api/runs/{id}` | 200 | нет |
| `GET` | `/api/runs/{id}/metrics` | 200 | нет → **JSON-массив** |
| `GET` | `/api/runs/{id}/snapshot` | 200 | query `t_s`, `client_id` |
| `GET` | `/api/runs/{id}/export` | 200 | нет |
| `POST` | `/api/runs/{id}/what-if` | **201** | `WhatIfRequest` |
| `POST` | `/api/compare` | 200 | `run_a`+`run_b` или `run_ids` |

Других маршрутов нет (нет списка, удаления, логина, websocket, прогресса, PUT).

---

## 6. Health

`GET /health` → **200**

```json
{"status":"ok"}
```

---

## 7. Проекты

### 7.1. Создать — `POST /api/projects` → 201

Два входа.

**А) Сценарий `cosmo-A-1.0`** (тело = scenario). Официальная фикстура:

```bash
curl -s -X POST http://localhost:8080/api/projects \
  -H 'Content-Type: application/json' \
  --data-binary @data/01_full_constellation.json
```

Минимально валидный пример (для юнит-тестов UI, не для демо суток):

```json
{
  "schema_version": "cosmo-A-1.0",
  "meta": { "id": "demo", "title": "Демо" },
  "environment": {
    "altitude_km": 550,
    "inclination_deg": 87,
    "earth_angle0_deg": 12,
    "horizon_s": 120,
    "step_s": 120,
    "min_elevation_deg": 10,
    "isl_range_km": 3000,
    "target_availability": 0.9
  },
  "design": {
    "launch_stage": 3,
    "planes": [{ "id": "P1", "raan_deg": 0, "phase_deg": 0 }],
    "satellites": [{ "id": "S01", "plane_id": "P1", "slot_deg": 0, "launch_batch": 1 }]
  },
  "ground_sites": [
    { "id": "C65", "name": "Northern terminal 65", "role": "client", "lat_deg": 65, "lon_deg": 60 },
    { "id": "G_MUR", "name": "Murmansk", "role": "gateway", "lat_deg": 68.97, "lon_deg": 33.07 }
  ],
  "failures": [],
  "gateway_outages": []
}
```

**Б) Выгрузка `cosmo-A-result-1.0`**: берётся только `effective_scenario`. `routes` / `metrics` для нового проекта **не** используются. Без `effective_scenario` → 400 (`effective_scenario required for cosmo-A-result-1.0`).

Ответ — **сам Project**, не `{project: ...}`:

```json
{
  "id": "3f2a0c1e-7b44-4c1a-9d2e-0a1b2c3d4e5f",
  "base": { "schema_version": "cosmo-A-1.0", "meta": { "id": "01_full_constellation", "title": "Полная группировка" } },
  "effective": { "schema_version": "cosmo-A-1.0", "meta": { "id": "01_full_constellation", "title": "Полная группировка" } }
}
```

- `id` проекта — новый UUID. `meta.id` фикстуры **не** становится ключом API.
- `base` и `effective` на старте одинаковые (полные копии сценария).
- Перед сохранением сервер считает geometry на `t_s=0`.

Валидация сценария (400):

| Поле | Ограничение |
|---|---|
| `schema_version` | ровно `cosmo-A-1.0` |
| `horizon_s`, `step_s` | > 0, `horizon_s % step_s == 0`, `step_s ≤ horizon_s ≤ 172800` |
| `altitude_km` | 200…1200 |
| `inclination_deg` | (0, 180] |
| `min_elevation_deg` | [0, 90) |
| `isl_range_km` | (0, 10000] |
| `target_availability` | [0, 1] |
| `launch_stage` | 1, 2 или 3 |
| `planes[].raan_deg` / `phase_deg` | конечные, **[0, 360)** (360 нельзя) |
| `satellites[].launch_batch` | 1…3, `plane_id` известен |
| `ground_sites[].role` | только `client` или `gateway`; нужен хотя бы один каждый |
| lat/lon | lat [-90, 90], lon [-180, 180] |
| `failures[].satellite_id` | существующий КА; интервал `0 ≤ start_s < end_s ≤ horizon_s` |
| `gateway_outages[].gateway_id` | существующий шлюз; тот же интервал |

Плоскости и КА обязательны, id уникальны, id пункта не должен совпадать с id КА.

### 7.2. Читать — `GET /api/projects/{id}` → 200

Тот же `Project`. 404 если нет.

### 7.3. Патч — `PATCH /api/projects/{id}` → 200

Меняет **только `effective`**. `base` не трогается. Нельзя менять environment, спутники, пункты, meta — **нет в API**. Уже созданные прогоны **не** пересчитываются.

```json
{
  "launch_stage": 1,
  "planes": [
    { "id": "P1", "raan_deg": 10.0, "phase_deg": 5.0 }
  ],
  "failures": [
    { "satellite_id": "S01", "start_s": 0, "end_s": 3600 }
  ],
  "gateway_outages": [
    { "gateway_id": "G_MUR", "start_s": 0, "end_s": 120 }
  ]
}
```

Семантика указателей:

| JSON | Эффект |
|---|---|
| поле отсутствует | не менять |
| `"failures": null` | **не менять** (в Go это nil-указатель) |
| `"failures": []` | **очистить** список |
| `"gateway_outages": []` | очистить outage шлюзов |
| `"planes": []` | ничего (пустой список плоскостей игнорируется) |
| неизвестный `planes[].id` | 400 `unknown plane id` |
| нельзя добавить/удалить плоскость | **нет в API** |

После патча сценарий заново валидируется и считается snapshot `t=0`. Неверные углы/stage — 400.

`{}` валиден: пересчёт без изменений, 200.

### 7.4. Сброс — `POST /api/projects/{id}/reset` → 200

`effective = clone(base)`. Уже созданные прогоны **не** удаляются (у них свой снимок сценария).

### 7.5. Копия — `POST /api/projects/{id}/copy` → **201**

Новый UUID, копии `base` и `effective`. Прогоны не копируются.

---

## 8. Прогоны

### 8.1. Создать — `POST /api/projects/{id}/runs` → **201**

Тело не нужно. Считается **effective** проекта на всей сетке. Для фикстур суток: `horizon_s=86400`, `step_s=120` → **720** шагов, маршруты = 720 × число клиентов (в офиц. фикстурах 3 клиента → 2160 записей).

Ответ — **не** полный `Run`:

```json
{
  "run_id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
  "project_id": "3f2a0c1e-7b44-4c1a-9d2e-0a1b2c3d4e5f",
  "metrics": [
    {
      "client_id": "C65",
      "visibility_ratio": 0.98,
      "path_ratio": 0.98,
      "max_gap_s": 120,
      "mean_hops": 3.1,
      "meets_target": true,
      "gaps": [
        { "start_s": 3600, "end_s": 3720, "duration_s": 120, "reason": "no_visible_sat" }
      ]
    }
  ]
}
```

`gaps` всегда массив. `reason` у интервала может отсутствовать, если доминантной причины нет.

Поля `summary` в этом ответе **нет** — оно есть в `GET /api/runs/{id}`. Порядок `metrics` = порядок клиентов в `ground_sites`.

404 если проекта нет. 422 если python/сетка сломались.

`PATCH` **не** пересчитывает уже созданные прогоны. После правок нужен новый `POST .../runs`. Старый run хранит свой `effective_scenario`.

### 8.2. Полный прогон — `GET /api/runs/{id}` → 200

Тяжёлый ответ (сценарий + все `routes` + `metrics`). Python **не** вызывается — маршруты уже в прогоне. Для дашборда KPI берите `/metrics`; этот GET — таймлайн без повторной геометрии.

```json
{
  "id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
  "project_id": "3f2a0c1e-7b44-4c1a-9d2e-0a1b2c3d4e5f",
  "effective_scenario": {},
  "routes": [
    { "t_s": 0, "client_id": "C65", "path": ["C65", "S01", "G_MUR"], "hops": 2 },
    { "t_s": 120, "client_id": "C65", "path": [], "reason": "isl_partition" }
  ],
  "metrics": [],
  "summary": "Все пункты достигают целевой доступности 90%"
}
```

Путь на разрыве — `[]`, не `null`. `hops` на разрыве нет. `path[0]` = клиент, последний узел = шлюз.

### 8.3. Метрики — `GET /api/runs/{id}/metrics` → 200

**Массив**, не `{metrics:[...]}`:

```json
[
  {
    "client_id": "C65",
    "visibility_ratio": 1,
    "path_ratio": 1,
    "max_gap_s": 0,
    "mean_hops": 2,
    "meets_target": true,
    "gaps": []
  }
]
```

`meets_target: false` **приходит**, не скрывается.

---

## 9. Снимок сети — `GET /api/runs/{id}/snapshot`

Query:

| Параметр | Тип | По умолчанию | Правила |
|---|---|---|---|
| `t_s` | float | `0` если пусто | конечное число, **`[0, horizon_s]` включительно**. Не число (`abc`) → 400 `t_s must be a finite number`. Вне диапазона → 400 `t_s must be within [0, horizon_s]` |
| `client_id` | string | первый клиент сценария | неизвестный → 400 `unknown client "..."` |

Имя параметра только `t_s`, не `t`. Сетка не обязательна: `t_s=60` при шаге 120 валиден; geometry считается в этой точке; `network_delta` смотрит **последний шаг сетки с `t < t_s`**.

```bash
curl -s 'http://localhost:8080/api/runs/{id}/snapshot?t_s=0&client_id=C65'
```

Ответ 200:

```json
{
  "snapshot": {
    "t_s": 0,
    "satellites": [
      { "id": "S01", "x_km": 1234.5, "y_km": -800.1, "z_km": 6400.2, "active": true }
    ],
    "edges": [
      ["C65", "S01", 812.4],
      ["S01", "G_MUR", 905.0]
    ],
    "elevation_deg": {
      "C65": { "S01": 27.5 },
      "G_MUR": { "S01": 15.2 }
    }
  },
  "route": {
    "path": ["C65", "S01", "G_MUR"],
    "hops": 2,
    "alternatives": [["C65", "S02", "G_MUR"]],
    "algorithm": {
      "name": "bfs_min_hops",
      "objective": "минимальное число hops от клиента до шлюза",
      "constraints": [
        "клиент соединяется только со спутником",
        "ISL только между активными спутниками",
        "шлюз — стоп маршрута",
        "наземные пункты не ретранслируют трафик"
      ],
      "rationale": "На каждом шаге сетки граф меняется целиком, поэтому допустимые маршруты ищутся заново BFS. Критерий — hops, а не километры: каждый hop это отдельный радиоканал и задержка. Запасные пути той же длины и +1 hop показывают обход при отказе узла.",
      "min_hops": 2
    }
  },
  "network_delta": {
    "changed": false,
    "previous_still_valid": false,
    "current_path": ["C65", "S01", "G_MUR"],
    "explanation": "Первый шаг сетки: маршрут построен с нуля BFS (min hops). Запасные пути: 1"
  },
  "visible_satellites": ["S01"]
}
```

`edges` — кортеж `[id_a, id_b, distance_km]`, не `{from,to}`.

`x_km`/`y_km`/`z_km` — Earth-fixed ECEF, км. Координат наземных пунктов в snapshot **нет** — берите `lat_deg`/`lon_deg` из `effective.ground_sites`.

`elevation_deg[пункт][КА]` — **все активные** КА, в том числе ниже `min_elevation_deg`. Список «видит сейчас» — `visible_satellites`, не ключи elevation.

`active: false` приходит явно. Неактивные КА в `satellites` есть, в `elevation_deg` и рёбрах — нет.

`GET /api/runs/{id}` уже содержит все `routes` сетки (без повторного Python). Snapshot нужен для 3D/рёбер/elevation в произвольный `t_s`.

На `t_s=0` предыдущего шага нет: `previous_path` скорее всего **нет в JSON**, `changed` и `previous_still_valid` будут `false` — это не «маршрут сломался». Смотрите `explanation`.

Если пути нет:

```json
{
  "route": {
    "path": [],
    "reason": "no_visible_sat",
    "algorithm": { "name": "bfs_min_hops", "objective": "…", "constraints": [], "rationale": "…" }
  },
  "network_delta": {
    "changed": false,
    "previous_still_valid": false,
    "explanation": "Первый шаг сетки: допустимого маршрута нет (нет видимого спутника). Алгоритм: BFS min hops, маршруты ищутся заново на каждом t"
  },
  "visible_satellites": []
}
```

`hops` / `alternatives` / `min_hops` отсутствуют (`omitempty`).

Каждый snapshot **заново** зовёт python (не кэш суточного прогона), кроме `previous_path` из сохранённых `routes`.

---

## 10. Выгрузка — `GET /api/runs/{id}/export` → 200

Заголовок:

```
Content-Disposition: attachment; filename="cosmo-A-result.json"
```

Тело:

```json
{
  "schema_version": "cosmo-A-result-1.0",
  "effective_scenario": {},
  "routes": [
    { "t_s": 0, "client_id": "C65", "path": ["C65", "S01", "G_MUR"], "hops": 2 },
    { "t_s": 120, "client_id": "C65", "path": [], "reason": "isl_partition" }
  ],
  "metrics": [
    {
      "client_id": "C65",
      "visibility_ratio": 0.98,
      "path_ratio": 0.98,
      "max_gap_s": 120,
      "mean_hops": 3,
      "meets_target": true,
      "gaps": []
    }
  ],
  "summary": "Все пункты достигают целевой доступности 90%"
}
```

`path` никогда не `null`. `reason` только на разрыве. Повторная загрузка этого JSON в `POST /api/projects` создаёт **новый** проект из `effective_scenario`.

---

## 11. Compare — `POST /api/compare` → 200

Тело, **один** из вариантов:

```json
{ "run_a": "<uuid>", "run_b": "<uuid>" }
```

```json
{ "run_ids": ["<uuid1>", "<uuid2>", "<uuid3>"] }
```

Если `run_ids.length >= 2`, список **главнее** `run_a`/`run_b`. Один id или только `run_a` → 400 `two or more run ids required`. Неизвестный run → 404.

Порядок в ответе = порядок во входе (`run_a` потом `run_b`, либо порядок `run_ids`).

### Ровно 2 прогона

```json
{
  "run_a_id": "aaa",
  "run_b_id": "bbb",
  "config_diff": {
    "launch_stage": { "a": 3, "b": 1 },
    "isl_range_km": { "a": 3000, "b": 2000 },
    "planes": [
      {
        "id": "P1",
        "a": { "raan_deg": 0, "phase_deg": 0 },
        "b": { "raan_deg": 10, "phase_deg": 5 }
      }
    ],
    "failures": { "a": [], "b": [{ "satellite_id": "S01", "start_s": 0, "end_s": 120 }] }
  },
  "variants": [
    {
      "run_id": "aaa",
      "project_id": "proj-a",
      "title": "Полная группировка",
      "launch_stage": 3,
      "planes": [{ "id": "P1", "raan_deg": 0, "phase_deg": 0 }],
      "clients_meeting_target": 3,
      "clients_total": 3,
      "mean_path_ratio": 0.98,
      "mean_max_gap_s": 40,
      "mean_hops": 3.1,
      "metrics": [],
      "summary": "Все пункты достигают целевой доступности 90%"
    }
  ],
  "clients": [
    {
      "client_id": "C65",
      "better": "a",
      "path_ratio_a": 0.98,
      "path_ratio_b": 0.19,
      "delta_path_ratio": -0.79,
      "visibility_ratio_a": 0.99,
      "visibility_ratio_b": 0.4,
      "max_gap_s_a": 120,
      "max_gap_s_b": 3600,
      "delta_max_gap_s": 3480,
      "meets_target_a": true,
      "meets_target_b": false,
      "by_run": [
        { "run_id": "aaa", "path_ratio": 0.98, "visibility_ratio": 0.99, "max_gap_s": 120, "mean_hops": 3.1, "meets_target": true },
        { "run_id": "bbb", "path_ratio": 0.19, "visibility_ratio": 0.4, "max_gap_s": 3600, "mean_hops": 2.0, "meets_target": false }
      ]
    }
  ],
  "recommendation": {
    "run_id": "aaa",
    "better": "a",
    "reason": "Рекомендуется «Полная группировка» (этап 3): 3 из 3 пунктов достигают цели 90%, средняя доступность пути 98%",
    "advantages": ["…"],
    "conditions": ["Цель: path_ratio ≥ 90% (target_availability=0.90) для заданных наземных пунктов"],
    "limitations": ["…"],
    "conclusion": "Итог: брать «Полная группировка» (этап 3). …"
  }
}
```

`better` при двух прогонах: `"a"` | `"b"` | `"tie"`.

`config_diff` ключи (только отличия): `launch_stage`, `altitude_km`, `inclination_deg`, `earth_angle0_deg`, `min_elevation_deg`, `isl_range_km`, `target_availability`, `horizon_s`, `step_s`, `failures`, `gateway_outages`, `planes`. Если конфиги совпали, поля `config_diff` может не быть.

`delta_*` = **b − a**.

`meets_target_b: false` присутствует.

Критерий победы варианта: больше `clients_meeting_target`, затем больше `mean_path_ratio`, затем меньше `mean_max_gap_s`. По клиенту: `meets_target`, затем `path_ratio`, затем меньший `max_gap_s`.

### N > 2 (`run_ids`)

- `run_a_id` / `run_b_id` / `config_diff` **нет**.
- `recommendation.better` и `clients[].better` — **UUID победителя** или `"tie"`, не `a`/`b`.
- поля `path_ratio_a` и т.д. всё равно приедут как **нули**. Для N≠2 читайте только `by_run`.

Тексты `reason` / `advantages` / `conditions` / `limitations` / `conclusion` — готовые русские строки для UI.

---

## 12. What-if — `POST /api/runs/{id}/what-if` → **201**

Создаёт **новый проект** (base = сценарий исходного прогона, effective = тот же + отказ) и **новый полный прогон**. Исходный проект/run не меняются. Это так же медленно, как `POST .../runs`.

Тело:

```json
{
  "client_id": "C65",
  "t_s": 0,
  "satellite_id": "S01",
  "gateway_id": "G_MUR",
  "start_s": 0,
  "end_s": 86400
}
```

Все поля кроме выбора цели опциональны.

| Поле | Смысл |
|---|---|
| `satellite_id` | отказать этот КА (должен существовать) |
| `gateway_id` | outage этого шлюза |
| оба | отказать и КА, и шлюз |
| ни того ни другого | взять **первый спутник на сохранённом пути** клиента в шаге `int(t_s)` (нужно точное совпадение с точкой сетки) |
| `client_id` | для автовыбора КА с пути; по умолчанию первый клиент |
| `t_s` | старт отказа, если нет `start_s`; также ключ поиска пути при автовыборе (`int`, не округление до сетки) |
| `start_s` / `end_s` | интервал отказа; дефолт `[t_s, horizon_s]`; нужно `0 ≤ start < end ≤ horizon` |

Автовыбор с `t_s=60` при сетке 0,120,… → 400 `no route for client … at t_s=60`. Для клика по пути на снимке передавайте `satellite_id` с `path[1]`.

Пустой путь в этом шаге → 400 `no satellite on path`.

Ответ (сжатый):

```json
{
  "original_run_id": "aaa",
  "project_id": "new-project-uuid",
  "run_id": "new-run-uuid",
  "failed_satellite_id": "S01",
  "metrics": [],
  "compare": {
    "run_a_id": "aaa",
    "run_b_id": "new-run-uuid",
    "recommendation": {
      "run_id": "aaa",
      "better": "a",
      "reason": "Рабочая конфигурация — исходный прогон: …",
      "advantages": [],
      "conditions": [],
      "limitations": [
        "run_b — прогон с искусственным отказом, а не альтернативная конструкция группировки"
      ],
      "conclusion": "Итог: не выбирать run_b как лучший конфиг. …"
    }
  },
  "analysis": {
    "failed_satellite_id": "S01",
    "interval": { "start_s": 0, "end_s": 86400 },
    "affected_clients": ["C65"],
    "preserved_clients": ["C70", "C72"],
    "clients": [
      {
        "client_id": "C65",
        "affected": true,
        "route_preserved": false,
        "path_before": ["C65", "S01", "G_MUR"],
        "path_after": [],
        "reason_after": "isl_partition",
        "path_ratio_before": 0.98,
        "path_ratio_after": 0.81,
        "delta_path_ratio": -0.17,
        "max_gap_s_before": 120,
        "max_gap_s_after": 600,
        "delta_max_gap_s": 480,
        "meets_target_before": true,
        "meets_target_after": false,
        "lost_steps": 12,
        "window_steps": 720,
        "window_path_before": 700,
        "window_path_after": 688,
        "dominant_gap_reason": "isl_partition"
      }
    ],
    "gap_reasons": {
      "before": { "no_visible_sat": 0, "no_gateway_contact": 0, "gateway_outage": 0, "isl_partition": 10 },
      "after": { "no_visible_sat": 0, "no_gateway_contact": 0, "gateway_outage": 0, "isl_partition": 40 },
      "delta": { "gateway_outage": 0, "isl_partition": 30, "no_gateway_contact": 0, "no_visible_sat": 0 }
    },
    "vulnerabilities": ["…"],
    "mitigations": ["…"],
    "summary": "Отказ спутник S01 затрагивает …"
  }
}
```

`compare.recommendation.better` **всегда `"a"`**, `run_id` = исходный прогон. Даже если отказ «не ухудшил» доступность, run_b не предлагается как рабочий конфиг. Первая limitation всегда про искусственный отказ.

`failed_gateway_id` появляется вместо/вместе с КА. Булевы `affected`, `meets_target_*`, `route_preserved` всегда в JSON.

---

## 13. Сценарии UI

**Загрузить → править → считать**

1. `POST /api/projects` с файлом из `data/`.
2. Храните `project.id` у себя.
3. Редактор: крутилки RAAN/phase, `launch_stage` 1–3, таблица `failures` / `gateway_outages` → `PATCH`. Клиенты/шлюзы, орбита, ISL, состав КА — read-only (смена = новый JSON в `POST /api/projects`).
4. «Сбросить» → `POST .../reset`. «Дублировать вариант» → `POST .../copy` (новый id).
5. «Рассчитать сутки» → `POST .../runs`, спиннер до 5 мин, запомните `run_id`.
6. Дашборд: `GET .../metrics` (массив). Цель: `meets_target`, столбцы `path_ratio`, `visibility_ratio`, `max_gap_s`, полоски `gaps[]`.

**Карта/3D на времени t**

1. Селектор клиента (`C65` / `C70` / `C72`) и слайдер `t_s` по сетке (шаг `effective.environment.step_s`).
2. `GET /snapshot?t_s=&client_id=`.
3. Рёбра из `snapshot.edges`, КА из `satellites` (`active` = тусклый если false).
4. Основной путь `route.path`, запасные `route.alternatives`.
5. Подсветка `visible_satellites`.
6. Баннер `network_delta.explanation`; не трактуйте `previous_still_valid=false` на первом шаге как аварию.

**Сравнить полную группировку с очередью 1**

1. Прогон A на `launch_stage=3`.
2. `PATCH { "launch_stage": 1 }`, прогон B.
3. `POST /api/compare` `{ "run_a": A, "run_b": B }`.
4. Показать `config_diff.launch_stage`, таблицу `clients` (`meets_target_a/b`!), блок `recommendation.*`.
5. Не затирайте run A: reset проекта не удаляет старые прогоны.

**Отказ аппарата с маршрута**

1. Snapshot → `path[1]` = КА.
2. `POST .../what-if` `{ "satellite_id": "S01", "t_s": 0 }`.
3. Не предлагайте `compare.recommendation` как выбор «лучшего дизайна»: `better` всегда `a`. Рисуйте `analysis`.

**Экспорт / реимпорт**

1. `GET .../export` → скачать `cosmo-A-result.json` (имя из Content-Disposition).
2. Тот же JSON → `POST /api/projects` → новый проект.

## 14. Официальные фикстуры

Каталог `data/`. Пункты везде: шлюз `G_MUR`, клиенты `C65`, `C70`, `C72`. Сетка суток: `horizon_s=86400`, `step_s=120`. Цель `target_availability=0.9`.

Ожидаемые порядки (`internal/service/run/live_test.go`, допуск ±0.02 на mean path_ratio):

| Файл | Что это | mean path_ratio | 90% |
|---|---|---|---|
| `01_full_constellation.json` | этап 3, ISL 3000 км | ~0.98 | **все** пункты |
| `02_first_launch.json` | этап 1 | ~0.19 | **никто** |
| `03_satellite_outages.json` | этап 3, 10 КА отказ с 21600 с (S31, S14, S48, S16, S26, S15, S08, S05, S32, S34) | ~0.81 | **никто** |
| `04_link_range.json` | этап 3, ISL **2000** км | ~0.68 | vis ≳ 0.97, path ниже vis, в routes есть `isl_partition` |

`meta.id` совпадает с именем файла без `.json`. Это не UUID проекта.

## 15. Нет в API

- список / поиск / удаление проектов и прогонов
- авторизация, пользователи, роли
- websocket / SSE / прогресс расчёта
- пагинация `GET /runs/{id}`
- смена `environment`, состава КА, ground sites через PATCH
- выбор алгоритма маршрута (всегда `bfs_min_hops`)
- именование прогона, теги, комментарии
- постоянное хранилище (нет Postgres/файлов)
- вызов geometry из браузера
- `PUT`, `DELETE` по ресурсам (CORS их декларирует, маршрутов нет)
- OpenAPI/Swagger
- eclipse / sunlight (функция есть в `geometry.py`, API её не отдаёт)
- xyz наземных пунктов в snapshot
- dirty-флаг «effective ≠ base» (сравнивайте на клиенте)
- смена `isl_range_km` / орбиты / состава КА через PATCH — только новый `POST /api/projects` с JSON

Нет отдельного «живого» канала: новый `t_s` = новый `GET /snapshot`.

## 16. TypeScript (поля как в JSON)

Опциональные (`?`) = `omitempty` или отсутствуют в части режимов.

```ts
type GapReason =
  | "no_visible_sat"
  | "no_gateway_contact"
  | "gateway_outage"
  | "isl_partition";

interface ApiError { error: string }

interface Meta { id: string; title: string }

interface Environment {
  altitude_km: number;
  inclination_deg: number;
  earth_angle0_deg: number;
  horizon_s: number;
  step_s: number;
  min_elevation_deg: number;
  isl_range_km: number;
  target_availability: number;
}

interface Plane { id: string; raan_deg: number; phase_deg: number }
interface Satellite { id: string; plane_id: string; slot_deg: number; launch_batch: number }
interface GroundSite { id: string; name: string; role: "client" | "gateway"; lat_deg: number; lon_deg: number }
interface Failure { satellite_id: string; start_s: number; end_s: number }
interface GatewayOutage { gateway_id: string; start_s: number; end_s: number }

interface Scenario {
  schema_version: "cosmo-A-1.0" | string;
  meta: Meta;
  environment: Environment;
  design: { launch_stage: number; planes: Plane[]; satellites: Satellite[] };
  ground_sites: GroundSite[];
  failures: Failure[];
  gateway_outages: GatewayOutage[];
}

interface Project { id: string; base: Scenario; effective: Scenario }

interface Patch {
  launch_stage?: number;
  planes?: { id: string; raan_deg?: number; phase_deg?: number }[];
  failures?: Failure[];      // [] очищает; omit/null — не трогать
  gateway_outages?: GatewayOutage[];
}

interface GapInterval {
  start_s: number;
  end_s: number;
  duration_s: number;
  reason?: GapReason;
}

interface ClientMetrics {
  client_id: string;
  visibility_ratio: number;
  path_ratio: number;
  max_gap_s: number;
  mean_hops: number;
  meets_target: boolean;
  gaps: GapInterval[];
}

interface RouteRecord {
  t_s: number;
  client_id: string;
  path: string[];
  reason?: GapReason;
  hops?: number;
}

interface Run {
  id: string;
  project_id: string;
  effective_scenario: Scenario;
  routes: RouteRecord[];
  metrics: ClientMetrics[];
  summary?: string;
}

interface CreateRunResponse {
  run_id: string;
  project_id: string;
  metrics: ClientMetrics[];
}

type EdgeTuple = [string, string, number];

interface Snapshot {
  t_s: number;
  satellites: { id: string; x_km: number; y_km: number; z_km: number; active: boolean }[];
  edges: EdgeTuple[];
  elevation_deg: Record<string, Record<string, number>>;
}

interface RouteResult {
  path: string[];
  reason?: GapReason;
  hops?: number;
  alternatives?: string[][];
  algorithm: {
    name: string;
    objective: string;
    constraints: string[];
    rationale: string;
    min_hops?: number;
  };
}

interface NetworkDelta {
  changed: boolean;
  previous_still_valid: boolean;
  previous_path?: string[];
  current_path?: string[];
  explanation: string;
}

interface GetSnapshotResponse {
  snapshot: Snapshot;
  route: RouteResult;
  network_delta: NetworkDelta;
  visible_satellites: string[];
}

interface ExportDocument {
  schema_version: "cosmo-A-result-1.0" | string;
  effective_scenario: Scenario;
  routes: RouteRecord[];
  metrics?: ClientMetrics[];
  summary?: string;
}

interface CompareRunsRequest {
  run_a?: string;
  run_b?: string;
  run_ids?: string[];
}

interface CompareVariant {
  run_id: string;
  project_id: string;
  title: string;
  launch_stage: number;
  planes: Plane[];
  clients_meeting_target: number;
  clients_total: number;
  mean_path_ratio: number;
  mean_max_gap_s: number;
  mean_hops: number;
  metrics: ClientMetrics[];
  summary?: string;
}

interface ClientRunMetric {
  run_id: string;
  path_ratio: number;
  visibility_ratio: number;
  max_gap_s: number;
  mean_hops: number;
  meets_target: boolean;
}

interface ClientDiff {
  client_id: string;
  better: "a" | "b" | "tie" | string; // UUID при N>2
  path_ratio_a: number;
  path_ratio_b: number;
  delta_path_ratio: number;
  visibility_ratio_a: number;
  visibility_ratio_b: number;
  max_gap_s_a: number;
  max_gap_s_b: number;
  delta_max_gap_s: number;
  meets_target_a: boolean;
  meets_target_b: boolean;
  by_run: ClientRunMetric[];
}

interface CompareRecommendation {
  run_id?: string;
  better: "a" | "b" | "tie" | string;
  reason: string;
  advantages: string[];
  conditions: string[];
  limitations: string[];
  conclusion: string;
}

interface CompareRunsResponse {
  run_a_id?: string;
  run_b_id?: string;
  config_diff?: Record<string, unknown>;
  variants: CompareVariant[];
  clients: ClientDiff[];
  recommendation: CompareRecommendation;
}

interface WhatIfRequest {
  client_id?: string;
  t_s?: number;
  satellite_id?: string;
  gateway_id?: string;
  start_s?: number;
  end_s?: number;
}

interface ResilienceClient {
  client_id: string;
  affected: boolean;
  route_preserved: boolean;
  path_before: string[];
  path_after: string[];
  reason_before?: GapReason;
  reason_after?: GapReason;
  path_ratio_before: number;
  path_ratio_after: number;
  delta_path_ratio: number;
  max_gap_s_before: number;
  max_gap_s_after: number;
  delta_max_gap_s: number;
  meets_target_before: boolean;
  meets_target_after: boolean;
  lost_steps: number;
  window_steps: number;
  window_path_before: number;
  window_path_after: number;
  dominant_gap_reason?: GapReason;
}

interface WhatIfResponse {
  original_run_id: string;
  project_id: string;
  run_id: string;
  failed_satellite_id?: string;
  failed_gateway_id?: string;
  metrics: ClientMetrics[];
  compare: CompareRunsResponse;
  analysis: {
    failed_satellite_id?: string;
    failed_gateway_id?: string;
    interval: { start_s: number; end_s: number };
    affected_clients: string[];
    preserved_clients: string[];
    clients: ResilienceClient[];
    gap_reasons: {
      before: Record<string, number>;
      after: Record<string, number>;
      delta: Record<string, number>;
    };
    vulnerabilities: string[];
    mitigations: string[];
    summary: string;
  };
}
```

`GET /metrics` декодируйте как `ClientMetrics[]`.
`POST /projects` и `GET/PATCH/reset/copy` — как `Project`.
`POST /runs` — как `CreateRunResponse`, не как `Run`.
