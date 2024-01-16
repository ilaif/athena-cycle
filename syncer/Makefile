ifeq ($(wildcard .env),.env)
    include .env
    export
endif

lint:
	poetry run ruff check .

format:
	poetry run ruff format .

migrate:
	poetry run alembic upgrade head

migrate-dry-run:
	poetry run alembic upgrade head --sql
