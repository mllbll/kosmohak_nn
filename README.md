# КосмоХакатон NN — backend

Сервис проектирования устойчивой спутниковой группировки. Один процесс, три слоя как в `space-manufacture`: **api → service → repository**. Геометрия считается эталонным `python/geometry.py` без правок.

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

## Запуск

```bash
pip install numpy
make tidy
make run
```

Проверка эталона:

```bash
python3 python/runner.py data/01_full_constellation.json 0
```

## API

| Метод | Путь |
|---|---|
| GET | `/health` |
| POST | `/api/projects` |
| GET | `/api/projects/{id}` |
| PATCH | `/api/projects/{id}` |
| POST | `/api/projects/{id}/runs` |
| GET | `/api/runs/{id}` |
| GET | `/api/runs/{id}/metrics` |
| GET | `/api/runs/{id}/snapshot?t_s=&client_id=` |
| GET | `/api/runs/{id}/export` |
| POST | `/api/compare` |

```bash
curl -s -X POST http://localhost:8080/api/projects \
  -H 'Content-Type: application/json' \
  --data-binary @data/01_full_constellation.json
```
