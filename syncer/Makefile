include .env

lint:
	poetry run ruff check .

migrate:
	poetry run alembic upgrade head

migrate-dry-run:
	poetry run alembic upgrade head --sql
