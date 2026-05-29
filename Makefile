.PHONY: run test test-race vet fmt compose-up compose-down compose-logs test-integration

run:
	@cd examples/go-commerce-api && DB_DSN=$${DB_DSN:-postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable} go run ./cmd/api

test:
	@cd examples/go-commerce-api && go test ./...

test-race:
	@cd examples/go-commerce-api && go test -race ./...

vet:
	@cd examples/go-commerce-api && go vet ./...

fmt:
	@cd examples/go-commerce-api && gofmt -w .

compose-up:
	@docker compose -f examples/docker-compose.yml up --build

compose-down:
	@docker compose -f examples/docker-compose.yml down

compose-logs:
	@docker compose -f examples/docker-compose.yml logs -f

test-integration:
	@docker compose -f examples/docker-compose.yml up -d --wait postgres redis nats
	@cd examples/go-commerce-api && INTEGRATION_TESTS=1 DB_DSN=$${DB_DSN:-postgres://commerce:commerce@localhost:5432/commerce?sslmode=disable} REDIS_ADDR=$${REDIS_ADDR:-localhost:6379} NATS_URL=$${NATS_URL:-nats://localhost:4222} go test -tags=integration ./...
