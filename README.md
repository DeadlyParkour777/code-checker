# Code Checker
Сервис проверки решений задач. Принимает файл с кодом, прогоняет тесты и возвращает результат.

## Сервисы
- **gateway** — HTTP API + Swagger, агрегирует gRPC вызовы
- **auth_service** — регистрация/логин, JWT
- **problem_service** — задачи и тест-кейсы
- **submission_service** — приём посылок
- **judge_service** — выполнение кода
- **result_service** — хранение/выдача истории посылок

## Старт

**Требования:** Docker + Docker Compose.  

1) Создать .env файлы для каждого сервиса и в корне по шаблонным .env.example.
Можно сделать это через скрипт или настроить вручную. <br>
Автоматически заполнить через скрипт можно командой:
```bash
./scripts/init-env.sh
```

2) Запуск:
```bash
docker compose up --build
```

## Endpoints
- Healthcheck: `GET http://localhost:8000`
- Swagger UI: `http://localhost:8000/swagger`
- OpenAPI: `http://localhost:8000/openapi.yaml`

Основные:
- `POST /auth/register`
- `POST /auth/login`
- `GET /problems`, `GET /problems/{problemID}`
- `POST /submissions` (multipart: `problem_id`, `language`, `code_file`)
- `GET /submissions/history`

## Поддерживаемые языки
- `go`
- `python`

## Структура репозитория
- `services/` — сервисы
- `proto/` — protobuf схемы
- `pkg/` — общий код + сгенерированный gRPC-код
- `migrations/` — миграции Postgres
- `docker-compose.yml` — конфигурация Docker Compose для запуска
