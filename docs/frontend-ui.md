# UI для фронтенда

SPA в этом репозитории **нет**. Здесь — экраны, состояние и запросы. HTTP-поля: [frontend.md](frontend.md). Код, который можно положить в UI как есть: [frontend-client.ts](frontend-client.ts).

API: `http://localhost:8080` (Docker) или тот же хост, что у `HTTP_ADDR`. CORS по умолчанию `*`, `credentials: 'include'` **нельзя**. Авторизации нет.

## 1. Состояние (только клиент)

Списка проектов и прогонов в API нет. Рестарт сервера стирает память. Держите UUID у себя.

Ключ `localStorage`: `kosmohak.session.v1`.

```ts
type Session = {
  project_id: string;
  run_id: string;
  client_id: string;          // C65 | C70 | C72
  t_s: number;
  compare_run_ids: string[];  // сохранённые прогоны для сравнения
};
```

После `POST /projects` и `POST .../runs` сразу пишите id. Если `GET` вернул 404 — сессия протухла (рестарт Docker): предложите загрузить фикстуру заново.

Грязный редактор: на клиенте сравните `effective` с `base` (`launch_stage`, `planes[].raan_deg/phase_deg`, `failures`, `gateway_outages`). Сервер dirty-флага не отдаёт.

## 2. Fetch

Таймаут сервера **5 минут**. Ставьте `AbortSignal` на 5+ мин для `POST .../runs` и `POST .../what-if`. Обрабатывайте **422** (`{"error":"..."}`) и **504** (часто без JSON). chi 404/405 — `text/plain`, не `{error}`.

`GET /metrics` — **массив**, не объект. Create/copy/run/what-if → **201**.

Не вызывайте `python/geometry.py` из браузера.

## 3. Макет

```
[файл фикстуры] [Рассчитать сутки] [Сброс] [Копия] [Экспорт]
[этап 1–3] [RAAN/phase плоскостей] [failures / gateway_outages]

KPI по GET /metrics (массив): path_ratio, meets_target, gaps[]

[клиент C65/C70/C72]
[карта / 3D]     [панель устойчивости | исследование параметра]
[таймлайн t_s + тики уникального пути]
```

Фикстуры для «открыть файл»: `data/01_full_constellation.json` … `04_link_range.json`. Тело целиком в `POST /api/projects`.

## 4. Карта и запасные пути

На каждом движении слайдера: `GET /api/runs/{run_id}/snapshot?t_s={t}&client_id={id}`.

Рисовать:

| Слой | Откуда | Как |
|---|---|---|
| КА | `snapshot.satellites` | ECEF км; `active: false` — тусклый; неактивных нет в рёбрах |
| Земля | `effective.ground_sites` | `lat_deg`/`lon_deg` (xyz пунктов в snapshot нет) |
| Рёбра сети | `snapshot.edges` | `[id_a, id_b, km]`, не `{from,to}` |
| Видимость | `visible_satellites` | не ключи `elevation_deg` (там все активные КА, в т.ч. ниже горизонта) |
| Основной путь | `route.path` | жирная линия; `path[0]` клиент, последний — шлюз |
| Запасные | `route.alternatives` | полупрозрачные; поля нет (`omitempty`), если запасных нет |
| Смена сети | `network_delta.explanation` | на `t_s=0` `previous_still_valid=false` — не авария |

Счётчик **«путей на этом шаге: N»**:

```ts
function pathCount(route: { path: string[]; alternatives?: string[][] }): number {
  if (!route.path.length) return 0;
  return 1 + (route.alternatives?.length ?? 0);
}
```

То же число по суткам без snapshot: `path.length ? 1 + rec.alt_count : 0` в `GET /runs/{id}`.

Клик по узлу:

- `path[0]` / последний — не what-if спутника.
- `path[1] … path[n-2]` — спутник → панель устойчивости.
- Шлюз можно отдельно отдать в `gateway_id`, для демо ТЗ достаточно клика по КА.

## 5. Таймлайн уникальных путей

Не из `/metrics` и не из `gaps[]`. После расчёта суток один раз `GET /api/runs/{id}` (Python не зовётся).

Для выбранного `client_id`:

```ts
function uniquePathTicks(routes: RouteRecord[], clientId: string): number[] {
  return routes
    .filter((r) => r.client_id === clientId && r.path.length > 0 && r.alt_count === 0)
    .map((r) => r.t_s);
}
```

Покрасьте эти `t_s` на слайдере (единственная точка отказа: основной путь есть, запасных нет). На разрыве (`path: []`, `alt_count: 0`) тик **не** ставьте — это gap, его уже показывают `metrics[].gaps`.

Сами линии запасных путей на карте — только из **текущего** snapshot (`route.alternatives`). Не дергайте 720 snapshot заранее.

Сетка: `t = 0, step_s, 2*step_s, …` пока `t < horizon_s` (правый конец **не** входит). Для фикстур: 720 шагов, `step_s=120`.

## 6. Панель устойчивости (what-if)

После клика по спутнику на `route.path`:

1. Показать id КА и интервал. Дефолт `start_s = t_s` слайдера, `end_s = horizon_s`. Пользователь может сузить; нужно `0 ≤ start < end ≤ horizon`.
2. Запрос (спиннер до 5 мин, это полный новый прогон):

```http
POST /api/runs/{run_id}/what-if
Content-Type: application/json

{ "satellite_id": "S01", "t_s": 3600, "start_s": 3600, "end_s": 86400 }
```

Всегда передавайте `satellite_id` с пути. Автовыбор без id ищет маршрут только в **точке сетки** `int(t_s)` — `t_s=60` при шаге 120 даст 400.

3. Коды: 201 + тело; 400 (неизвестный КА, пустой путь, кривой интервал); 404; 422/504 как у обычного прогона.

4. Рисовать **только `analysis`**:

| Поле | UI |
|---|---|
| `summary` | заголовок панели |
| `interval` | «отказ с … по …» |
| `affected_clients` / `preserved_clients` | два списка |
| `clients[]` | таблица: `client_id`, `path_before`, `path_after`, `reason_after`, `delta_path_ratio`, `delta_max_gap_s`, `meets_target_before` → `meets_target_after`, `lost_steps` |
| `vulnerabilities` | список рисков |
| `mitigations` | список мер |
| `gap_reasons.delta` | опционально, сдвиг причин разрыва |

`path_after: []` — путь пропал. Булевы `affected`, `route_preserved`, `meets_target_*` приходят всегда, в том числе `false`.

5. **Не показывать** `compare.recommendation` как выбор лучшей группировки. `better` **всегда `"a"`**: run_b — искусственный отказ, не дизайн. `limitations[0]` это прямо говорит. Блок compare можно спрятать целиком.

6. Ответ содержит новые `project_id` и `run_id`. Исходные id не затирайте. Что-if **не** патчит текущий проект.

## 7. Исследование параметра (environment)

PATCH меняет только `launch_stage`, RAAN/phase существующих плоскостей, `failures`, `gateway_outages`. **Нельзя** PATCH-ем сменить `isl_range_km`, высоту, наклонение, `min_elevation_deg`, состав КА, пункты.

Сценарий:

1. `structuredClone(project.effective)`.
2. Поменять **одно** поле `environment` (для демо: `isl_range_km: 2000`).
3. Углы плоскостей перед отправкой: `[0, 360)` — 360 нельзя, заворачивайте, не клэмпьте в 360.
4. `POST /api/projects` этим JSON → новый UUID.
5. `POST /api/projects/{новый}/runs` (снова до 5 мин).
6. `POST /api/compare` `{ "run_a": "<исходный>", "run_b": "<новый>" }` — только с одинаковыми `horizon_s`/`step_s`, иначе 400.
7. Показать `config_diff` (ожидается `isl_range_km`), таблицу `clients` (`meets_target_a` / `meets_target_b` приходят и при `false`), тексты `recommendation.*`.

Ориентир: фикстура `04_link_range.json` — vis ~0.98–1.0, path ~0.68, в routes есть `isl_partition`.

Смена этапа запуска — это PATCH `launch_stage`, не новый POST (см. compare 01 vs 02 в frontend.md §13).

## 8. Редактор и KPI

- RAAN/phase: wrap в `[0, 360)`, затем PATCH `{ "planes": [{ "id": "P1", "raan_deg": 10 }] }`. Нельзя добавить плоскость.
- `launch_stage`: 1, 2 или 3.
- `failures: []` очищает; отсутствие поля / `null` — не трогать.
- После PATCH старый run **не** пересчитывается. Нужен новый `POST .../runs`. Старый run хранит свой `effective_scenario`.
- Сброс: `POST .../reset` (`effective = base`). Прогоны не удаляются.
- KPI: `GET .../metrics` → массив. Цель: `meets_target` ⇔ `path_ratio >= target_availability` (в фикстурах 0.9). Полоски разрывов — `gaps[]` (`[start_s, end_s)`).
- Экспорт: `GET .../export` (`cosmo-A-result-1.0`, имя из `Content-Disposition`). Поля `alt_count` в выгрузке нет. Тот же JSON снова в `POST /api/projects` — берётся `effective_scenario`, маршруты выгрузки **не** импортируются.

## 9. Чеклист ловушек

- Нет списка проектов; 404 после рестарта → новая загрузка.
- Нет SSE/прогресса — спиннер.
- `GET /metrics` = JSON array.
- На gap: `path: []`, нет `hops`, есть `reason` и `alt_count: 0`.
- Snapshot `alternatives` нет в JSON, если запасных нет; в прогоне `alt_count` всегда есть.
- `t_s` query: конечное число в `[0, horizon_s]`; имя параметра `t_s`, не `t`.
- Неизвестный `client_id` → 400.
- What-if `compare.better` всегда `"a"` — не заголовок «лучший вариант».
- Не `credentials: 'include'` при CORS `*`.
- `x_km,y_km,z_km` — ECEF км; землю брать из сценария.
