network:
	sudo docker network create neka_pay_network

network.rm:
	sudo docker network rm neka_pay_network || true

postgres.stop:
	sudo docker stop postgres-go || true
	sudo docker rm postgres-go || true

postgres: postgres.stop
	sudo docker run --name postgres-go --network neka_pay_network -p 5433:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -d postgres:14-alpine

createdb:
	sudo docker exec -it postgres-go createdb --username=root --owner=root neka_pay

dropdb:
	sudo docker exec -it postgres-go dropdb neka_pay

migrateup:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose up
migrateup1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose up 1

migratedown:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose down
migratedown1:
	migrate -path db/migration -database "postgresql://root:secret@localhost:5433/neka_pay?sslmode=disable" -verbose down 1

sqlc:
	sqlc generate
test:
	go test -v -cover ./...
server:
	go run main.go
postgres-ready:
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in 1 2 3 4 5; do \
		if sudo docker exec postgres-go pg_isready -U root > /dev/null 2>&1; then \
			echo "PostgreSQL is ready!"; \
			exit 0; \
		fi; \
		echo "PostgreSQL is not ready yet. Waiting..."; \
		sleep 2; \
	done; \
	echo "PostgreSQL did not become ready in time!"; \
	exit 1

setup: postgres postgres-ready createdb migrateup

clean: postgres.stop network.rm

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 sqlc test setup server network network.rm clean