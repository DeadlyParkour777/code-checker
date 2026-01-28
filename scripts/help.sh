#!/usr/bin/env bash
set -euo pipefail

cat <<'HELP'
Доступные команды:
  make env         - создать .env файлы из .env.example
  make build       - собрать все сервисы
  make test        - запустить тесты во всех модулях
  make test-v      - тесты с подробным выводом
  make test-quiet  - подробный вывод только по тестам (без лишних логов)
  make cover       - покрытие по каждому пакету (в консоли)
  make cover-quiet - покрытие только там, где оно есть
  make tidy        - go mod tidy во всех модулях
  make fmt         - gofmt для всех Go файлов
  make compose-up  - docker compose up --build
  make compose-down - docker compose down
  make compose-build - docker compose build
  make compose-logs  - docker compose logs -f
HELP
