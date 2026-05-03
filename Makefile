.PHONY: run build migrate migrate-down docker-up docker-down tidy

run:
	go run ./cmd/server

build:
	go build -o bin/bilimquiz ./cmd/server

tidy:
	go mod tidy

migrate:
	@echo "Running migrations..."
	@for f in migrations/*.sql; do \
		echo "Applying $$f"; \
		psql "$$DB_URL" -f "$$f"; \
	done

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f app

db-shell:
	docker compose exec postgres psql -U postgres -d bilimquiz
