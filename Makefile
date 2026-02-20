.PHONY: run-example test-examples compose-up compose-down compose-logs

run-example:
	@cd examples/go-commerce-api && go run ./cmd/api

test-examples:
	@cd examples/go-commerce-api && go test ./...

compose-up:
	@docker compose -f examples/docker-compose.yml up --build

compose-down:
	@docker compose -f examples/docker-compose.yml down

compose-logs:
	@docker compose -f examples/docker-compose.yml logs -f
