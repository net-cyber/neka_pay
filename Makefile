network:
	sudo docker network create neka_pay-network

postgres.stop:
	sudo docker stop postgres-go || true
	sudo docker rm postgres-go || true

postgres: postgres.stop
	sudo docker run --name postgres-go --network neka_pay-network -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	sudo docker exec -it postgres-go createdb --username=root --owner=root neka_pay

dropdb:
	sudo docker exec -it postgres-go dropdb neka_pay

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose down

sqlc:
	sqlc generate

.PHONY: postgres createdb dropdb migrateup migratedown sqlc