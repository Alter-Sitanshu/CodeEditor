include .env
MIGRATION_PATH = "./cmd/migrate/migrations"

.PHONY: createdb
createdb:
	docker run -d --name code-editor-db -e POSTGRES_USER=${DB_USER} -e POSTGRES_PASSWORD=${DB_PASS} -p 5432:5432 postgres:12-alpine

.PHONY: startdb
startdb:
	docker run code-editor-db

.PHONY: migrate_up
migrate_up:
	@migrate -path $(MIGRATION_PATH) -database $(DB_ADDR) -verbose up

.PHONY: migrate_down
migrate_down:
	@migrate -path $(MIGRATION_PATH) -database $(DB_ADDR) -verbose down

.PHONY: migrations
migrations:
	@migrate create -seq -ext sql -dir $(MIGRATION_PATH) $(filter-out $@, $(MAKECMDGOALS))
