ifeq ($(wildcard .env),.env)
    include .env
    export
endif

lint:
	golangci-lint run ./... --fix

migrate:
	go run ./cmd/migrate

migrate-down:
	go run ./cmd/migrate down
