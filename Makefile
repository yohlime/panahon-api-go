-include app.env

createdb:
	docker exec -it pg12 createdb --username=${POSTGRES_ADMIN_USER} --owner=${POSTGRES_DB_USER} ${POSTGRES_DB}
	docker exec -it pg12 psql ${POSTGRES_DB} -U ${POSTGRES_ADMIN_USER} -c "CREATE EXTENSION postgis;"

dropdb:
	docker exec -it pg12 dropdb --username=${POSTGRES_ADMIN_USER} ${POSTGRES_DB}

migrateup:
	migrate -path internal/db/migration -database "${DB_SOURCE}" -verbose up

migrateup1:
	migrate -path internal/db/migration -database "${DB_SOURCE}" -verbose up 1

migratedown:
	migrate -path internal/db/migration -database "${DB_SOURCE}" -verbose down

migratedown1:
	migrate -path internal/db/migration -database "${DB_SOURCE}" -verbose down 1

new_migration:
	migrate create -ext sql -dir internal/db/migration -seq $(name)

sqlc:
	sqlc generate

server:
	go run cmd/api/main.go

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


.PHONY: createdb dropdb migrateup migrateup1 migratedown migratedown1 new_migration sqlc server mock swag test short_test build
