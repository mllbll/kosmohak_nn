# Бэкенд: запуск, настройка и работа с API

Инструкция для Go API и расчётного модуля Python. Все команды выполняются из корня репозитория, если не указано иное. Node.js и сборка фронтенда для работы бэкенда не нужны.

## Запуск только бэкенда в Docker

Требуется Docker с Docker Compose:

```bash
docker compose up -d --build api
docker compose ps api
docker compose logs -f api
```

API доступен на `http://localhost:8080`. Эти команды запускают только сервис `api`; уже работающие сервисы не останавливаются.

Проверка работоспособности:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Ожидаемый ответ: `{"status":"ok"}`. Для Bash используйте `curl http://localhost:8080/health`.

Остановка и повторный запуск:

```bash
docker compose stop api
docker compose start api
```

После изменения исходников снова выполните `docker compose up -d --build api`.

### Без Compose

```bash
docker build -t kosmohak-nn-api .
docker run --rm --name kosmohak-api -p 8080:8080 kosmohak-nn-api
```

Контейнер работает в текущем терминале; остановка — `Ctrl+C`. Не запускайте одновременно два контейнера на одном порту. Другой внешний порт можно задать через `-p 8081:8080`.

Корневой `Dockerfile` собирает Go-бинарник и включает Python, NumPy, `python/` и `data/`. Встроенная проверка здоровья обращается к внутреннему порту 8080; для смены внешнего порта достаточно изменить публикацию порта.

## Запуск из исходников

Требуются Go 1.23+ и Python 3.10+ с NumPy. Используйте виртуальное окружение Python.

### Windows PowerShell

```powershell
python -m venv .venv
.\.venv\Scripts\python.exe -m pip install numpy
go mod download

$env:PYTHON_BIN = (Resolve-Path .\.venv\Scripts\python.exe).Path
$env:RUNNER_SCRIPT = "python/runner.py"
$env:HTTP_ADDR = ":8080"
go run ./cmd
```

### Linux / macOS, Bash

```bash
python3 -m venv .venv
.venv/bin/python -m pip install numpy
go mod download

export PYTHON_BIN="$PWD/.venv/bin/python"
export RUNNER_SCRIPT="python/runner.py"
export HTTP_ADDR=":8080"
go run ./cmd
```

Сервис работает в текущем терминале; остановка — `Ctrl+C`.

Для отдельной сборки:

```bash
go build -o bin/api ./cmd
```

На Windows используйте `go build -o bin/api.exe ./cmd`. Собранному бинарнику по-прежнему нужны Python с NumPy и доступный `RUNNER_SCRIPT`; один бинарник не содержит расчётный модуль.

## Переменные окружения

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `HTTP_ADDR` | `:8080` | Адрес и порт HTTP-сервера. |
| `PYTHON_BIN` | `python3` | Команда или полный путь к Python с установленным NumPy. |
| `RUNNER_SCRIPT` | `python/runner.py` | Путь к обёртке расчётного модуля; относительный путь считается от рабочей директории процесса. |
| `CORS_ORIGINS` | `*` | Разрешённые источники HTTP-запросов, несколько значений разделяются запятыми. |

Настройки читает [internal/config/config.go](../internal/config/config.go). При запуске через Compose значения берутся из секции `services.api.environment` в [docker-compose.yml](../docker-compose.yml).

## Проверка расчётного модуля

Отдельный снимок полной группировки в момент `t = 0`:

```powershell
# PowerShell, после создания .venv
& $env:PYTHON_BIN python/runner.py data/01_full_constellation.json 0
```

```bash
# Bash, после настройки PYTHON_BIN
"$PYTHON_BIN" python/runner.py data/01_full_constellation.json 0
```

Команда должна вывести JSON с координатами спутников, рёбрами связей и углами места. Без последнего аргумента `runner.py` выводит снимки всей временной сетки, по одному JSON на строку.

Проверка внутри запущенного контейнера:

```bash
docker compose exec api python3 python/runner.py data/01_full_constellation.json 0
```

## Пример работы через API

Ниже последовательный пример для PowerShell. Он создаёт проект, рассчитывает его, получает снимок на дробную секунду и сохраняет экспорт.

```powershell
$apiBase = "http://localhost:8080"
$scenarioJson = Get-Content -Raw -Encoding UTF8 data/01_full_constellation.json
$scenarioBytes = [System.Text.Encoding]::UTF8.GetBytes($scenarioJson)

$project = Invoke-RestMethod -Method Post -Uri "$apiBase/api/projects" -ContentType "application/json; charset=utf-8" -Body $scenarioBytes
$run = Invoke-RestMethod -Method Post -Uri "$apiBase/api/projects/$($project.id)/runs"
$runId = $run.run_id

Invoke-RestMethod "$apiBase/api/runs/$runId/metrics"
Invoke-RestMethod "$apiBase/api/runs/$runId/snapshot?t_s=1.5&client_id=C65"
Invoke-WebRequest "$apiBase/api/runs/$runId/export" -OutFile result.json
```

Сравнение с первой очередью запуска:

```powershell
Invoke-RestMethod -Method Patch -Uri "$apiBase/api/projects/$($project.id)" -ContentType "application/json" -Body '{"launch_stage":1}'
$secondRun = Invoke-RestMethod -Method Post -Uri "$apiBase/api/projects/$($project.id)/runs"
$comparisonBody = @{ run_a = $runId; run_b = $secondRun.run_id } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$apiBase/api/compare" -ContentType "application/json" -Body $comparisonBody
```

Каждый запуск сохраняет конфигурацию на момент расчёта. Изменение проекта не меняет уже созданный прогон.

### Основные endpoints

| Метод | Путь | Назначение |
|---|---|---|
| GET | `/health` | Работоспособность HTTP-сервера. |
| POST | `/api/projects` | Создать проект из сценария или экспорта. |
| GET / PATCH | `/api/projects/{id}` | Получить или изменить проект. |
| POST | `/api/projects/{id}/reset` | Вернуть исходную конфигурацию. |
| POST | `/api/projects/{id}/copy` | Создать копию проекта. |
| POST | `/api/projects/{id}/runs` | Выполнить расчёт. |
| GET | `/api/runs/{id}` | Получить прогон. |
| GET | `/api/runs/{id}/metrics` | Получить показатели по клиентским пунктам. |
| GET | `/api/runs/{id}/snapshot?t_s=1.5&client_id=C65` | Рассчитать снимок сети и маршрут на указанное время. |
| GET | `/api/runs/{id}/export` | Экспортировать результат. |
| POST | `/api/runs/{id}/what-if` | Рассчитать последствия отказа спутника или шлюза. |
| POST | `/api/compare` | Сравнить прогоны. |

Полные JSON-схемы, тела запросов, ответы и коды ошибок: [контракт HTTP API](frontend.md).

## Правила расчёта и входных данных

- Сценарий использует `cosmo-A-1.0`, экспорт — `cosmo-A-result-1.0`. При импорте экспорта создаётся новый проект из `effective_scenario`; сохранённые маршруты не становятся новым прогоном.
- `horizon_s` и `step_s` — положительные целые секунды. Горизонт не превышает 172800 секунд и делится на шаг без остатка.
- Итоговые метрики считаются на сетке `0, step_s, 2 × step_s, … < horizon_s`. Начальный и конечный перерывы не объединяются.
- Endpoint снимка принимает произвольное время, включая дробные секунды, в пределах `[0, horizon_s]`. Оно не округляется к сетке метрик.
- Интервалы отказов имеют вид `[start_s, end_s)`. Не запущенные и отказавшие спутники исключаются из графа связей.
- Маршрутизация минимизирует число рёбер, включая две наземные линии. Клиентские пункты не служат транзитом, шлюз завершает путь.
- При отсутствии маршрута `path` — пустой массив, `hops` отсутствует; причина передаётся в `reason`.
- Сравнение требует разных ID прогонов, одинаковых периода, шага, цели доступности, состава и координат клиентов. Изменение орбит и дальности ISL допускается.

Валидация: [internal/service/project/validate.go](../internal/service/project/validate.go). Геометрия: [python/geometry.py](../python/geometry.py). Маршрутизация и метрики: [internal/service/run/](../internal/service/run/).

## Тестирование

### Полный набор в Docker

Окружение содержит Go, Python, NumPy и инструменты для проверки гонок:

```powershell
docker build -f Dockerfile.verify -t kosmohak-tests .
docker run --rm --mount "type=bind,source=$PWD,target=/src" kosmohak-tests
```

В Bash:

```bash
docker build -f Dockerfile.verify -t kosmohak-tests .
docker run --rm --mount "type=bind,source=$(pwd),target=/src" kosmohak-tests
```

Выполняются Python-тесты геометрии и `go test -race -count=1 ./...`. В этом окружении отсутствие расчётного модуля считается ошибкой, а не причиной пропуска интеграционных тестов.

### Локально

```bash
python3 -m unittest discover -s python -v
REQUIRE_GEOMETRY_TESTS=1 go test -race -count=1 ./...
```

Интеграционные Go-тесты запускают команду `python3` напрямую, независимо от `PYTHON_BIN`. Она должна быть доступна в `PATH` и импортировать NumPy. Для `-race` нужен поддерживаемый Go C-компилятор. На Windows удобнее использовать проверочное Docker-окружение.

Тесты охватывают валидацию, API, маршруты, метрики, отказы, сравнение, экспорт и четыре суточных сценария из [data/](../data/).

## Структура бэкенда

```text
cmd/main.go            Запуск процесса и корректное завершение
internal/api/          HTTP-обработчики
internal/app/          Сборка зависимостей, маршруты и HTTP-сервер
internal/config/       Переменные окружения
internal/model/        Сценарии, проекты, прогоны, запросы и ответы
internal/service/      Валидация, маршрутизация, метрики и сравнение
internal/repository/   Хранение проектов и прогонов в памяти
internal/client/       Вызов Python-модуля
python/geometry.py     Геометрия орбит и контактов
python/runner.py       Получение одного снимка или всей сетки
```

## Эксплуатация и диагностика

Проекты и прогоны хранятся в памяти одного процесса. Перезапуск удаляет их; сохраняйте нужные результаты через экспорт. Несколько экземпляров API не разделяют состояние. Авторизация и постоянная база данных не реализованы.

Расчёты синхронные: ответ приходит после завершения операции. HTTP middleware задаёт тайм-аут запроса 5 минут. Время расчёта зависит от сценария и машины.

| Симптом | Что проверить |
|---|---|
| Порт 8080 занят | Остановите другой процесс либо измените внешний порт контейнера. |
| Python не найден | Проверьте `PYTHON_BIN`; на Windows укажите полный путь к интерпретатору. |
| `No module named numpy` | Установите NumPy именно в интерпретатор, указанный в `PYTHON_BIN`. |
| Не найден `runner.py` | Запускайте из корня репозитория либо задайте абсолютный `RUNNER_SCRIPT`. |
| HTTP 400 при импорте | Прочитайте поле `error`: проверьте формат, диапазоны, ссылки на узлы и интервалы. |
| HTTP 404 для старого ID | После перезапуска сервера заново создайте проект и прогон. |
| Сравнение отклонено | Проверьте совместимость временной сетки, цели и клиентских пунктов. |
| API здоров, но расчёт не проходит | Посмотрите `docker compose logs api` и отдельно запустите `python/runner.py`. |
