-include app.env

createdb:
	${DOCKER_PG} createdb --username=${PG_ADMIN_USER} --owner=${PG_DB_USER} ${PG_DB_NAME}
	${DOCKER_PG} psql ${PG_DB_NAME} -U ${PG_ADMIN_USER} -c "CREATE EXTENSION postgis;"

dropdb:
	${DOCKER_PG} dropdb --username=${PG_ADMIN_USER} ${PG_DB_NAME}

MIGRATE_CMD = migrate -path ${MIGRATION_PATH} -database "${DB_SOURCE}" -verbose

migrate-cmd:
	@read -p "Enter the number of migrations to $(NAME) (leave empty for all): " count; \
	if [ -z "$$count" ]; then \
		${MIGRATE_CMD} $(ACTION); \
	else \
		${MIGRATE_CMD} $(ACTION) $$count; \
	fi

migrateup:
	$(MAKE) migrate-cmd NAME="apply" ACTION=up 

migratedown:
	$(MAKE) migrate-cmd NAME="roll back" ACTION=down

migratenew:
	migrate create -ext sql -dir ${MIGRATION_PATH} -seq $(name)

sqlc:
	sqlc generate

server:
	go run cmd/server/main.go

mock:
	mockery

swag:
	swag fmt -d cmd/server/main.go,internal/handlers
	swag init -o internal/docs/api -d cmd/server,internal/handlers,internal/models

test:
	go test -v -cover ./...

short_test:
	go test $(shell go list ./... | grep -v /db/) -cover -short 

build:
	go build -o main main.go


.PHONY: createdb dropdb migrateup migratedown migratenew sqlc server mock swag test short_test build
