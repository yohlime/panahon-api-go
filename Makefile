-include app.env

createdb:
	# createuser -h ${PG_HOST} -U ${PG_ADMIN_USER} --login --pwprompt ${PG_DB_USER}
	# createdb -h ${PG_HOST} -U ${PG_ADMIN_USER} --owner=${PG_DB_USER} ${PG_DB_NAME}
	psql -h ${PG_HOST} -p ${PG_PORT} ${PG_DB_NAME} -U ${PG_ADMIN_USER} -c "CREATE EXTENSION postgis;"

dropdb:
	dropdb --username=${PG_ADMIN_USER} ${PG_DB_NAME}

MIGRATE_CMD = migrate -path ${MIGRATION_PATH} -database "${DB_SOURCE}" -verbose

migrate-cmd:
	echo ${DB_SOURCE}
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
	go test -p 1 -v -cover ./...

e2e:
	go test -v -count=1 ./internal/db/e2e

short_test:
	go test $(shell go list ./... | grep -v /db/) -cover -short 

build:
	go build -o main main.go


.PHONY: createdb dropdb migrateup migratedown migratenew sqlc server mock swag test e2e short_test build
