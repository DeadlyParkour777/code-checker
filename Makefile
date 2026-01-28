.PHONY: help env build test test-v test-quiet cover cover-quiet tidy fmt compose-up compose-down compose-build compose-logs

help:
	@./scripts/help.sh

env:
	@./scripts/init-env.sh

build:
	@./scripts/build.sh

test:
	@./scripts/test.sh

test-v:
	@./scripts/test-v.sh

test-quiet:
	@./scripts/test-quiet.sh

cover:
	@./scripts/cover.sh

cover-quiet:
	@./scripts/cover-quiet.sh

tidy:
	@./scripts/tidy.sh

fmt:
	@./scripts/fmt.sh

compose-up:
	@docker compose up --build

compose-down:
	@docker compose down

compose-build:
	@docker compose build

compose-logs:
	@docker compose logs -f
