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

docker-up:
	sudo docker-compose down || true
	sudo docker stop postgres-go || true
	sudo docker rm postgres-go || true
	sudo docker-compose up -d

docker-down:
	sudo docker-compose down

docker-logs:
	sudo docker-compose logs -f

docker-migrate:
	sudo docker exec -it neka_pay-app migrate -path /app/db/migration -database "postgresql://root:secret@postgres-go:5432/neka_pay?sslmode=disable" -verbose up

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 sqlc test setup server docker-up docker-down docker-logs docker-migrate