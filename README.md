# Code Checker
Сервис проверки решений задач. Принимает файл с кодом, прогоняет тесты и возвращает результат.

## Сервисы
- **gateway** - HTTP API + Swagger, агрегирует gRPC вызовы
- **auth_service** - регистрация/логин, JWT
- **problem_service** - задачи и тест-кейсы
- **submission_service** - приём посылок
- **judge_service** - выполнение кода
- **result_service** - хранение/выдача истории посылок

## Старт

**Требования:** Docker + Docker Compose.  

1) Создать .env файлы для каждого сервиса и в корне по шаблонным .env.example:
```bash
make env
```

2) Запуск:
```bash
make compose-up
```

## Вспомогательные команды по проекту
```bash
make help
```

## Тесты
```bash
make test       # тесты (без подробного вывода)
make test-v     # тесты с подробным выводом
make test-quiet # подробный вывод только по тестам
make cover      # покрытие по каждому пакету
make cover-quiet # покрытие только там, где оно есть
```

Интеграционные тесты БД используют Testcontainers. Нужен запущенный Docker.

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
- `services/` - сервисы
- `proto/` - protobuf схемы
- `pkg/` - общий код + сгенерированный gRPC-код
- `migrations/` - миграции Postgres
- `docker-compose.yml` - конфигурация Docker Compose для запуска
