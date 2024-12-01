.PHONY: run compose-up migrate-up migrate-down

DATABASE_URL=postgres://postgres:admin@order-service-db:5432/postgres?sslmode=disable

run:
	docker-compose -f docker-compose.yaml up --build

migrate-up:
	@echo "Attempting to apply migrations up..."
	@for i in 1 2 3; do \
		echo "Migration attempt $$i..."; \
		docker-compose run --rm migrator -path /migrations -database ${DATABASE_URL} up && break || sleep 3; \
	done

migrate-down:
	@echo "Attempting to apply migrations down..."
	@for i in 1 2 3; do \
		echo "Migration attempt $$i..."; \
		docker-compose run --rm migrator -path /migrations -database ${DATABASE_URL} down && break || sleep 3; \
	done

lint:
	golangci-lint run