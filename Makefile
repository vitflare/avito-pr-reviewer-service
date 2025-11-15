# data from .env file
include .env
export $(shell sed 's/=.*//' .env)

DB_DRIVER=postgres
DB_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)
MIGRATIONS_DIR=migrations

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

run:
	@echo "Running application..."
	go run ./cmd/main.go

install-tools:
	@echo "Installing development tools..."
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed successfully!"

install-golangci-lint:
	@echo "$(GREEN)Installing golangci-lint...$(NC)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.6.1
	@echo "$(GREEN)golangci-lint installed successfully!$(NC)"

update-docs:
	@echo "Generating Swagger documentation..."
	swag init -g cmd/main.go -o ./docs

lint:
	@echo "Running linter..."
	golangci-lint run --config .golangci.yml

lint-fix:
	@echo "Running linter with auto-fix..."
	golangci-lint run --config .golangci.yml --fix

# goose commands
up:
	@echo "Applying migrations from $(MIGRATIONS_DIR)..."
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_URL)" up

down:
	@echo "Rolling back migrations..."
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_URL)" down

status:
	@echo "Migration status in $(MIGRATIONS_DIR):"
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(DB_URL)" status

# test db command
TEST_POSTGRES_USER=$(shell grep POSTGRES_USER tests/.env.tests | cut -d '=' -f2)
TEST_POSTGRES_PASSWORD=$(shell grep POSTGRES_PASSWORD tests/.env.tests | cut -d '=' -f2)
TEST_POSTGRES_HOST=$(shell grep POSTGRES_HOST tests/.env.tests | cut -d '=' -f2)
TEST_POSTGRES_PORT=$(shell grep POSTGRES_PORT tests/.env.tests | cut -d '=' -f2)
TEST_POSTGRES_SSL=$(shell grep POSTGRES_SSL_MODE tests/.env.tests | cut -d '=' -f2 || echo disable)
TEST_POSTGRES_DB=pr_reviewer_test
TEST_DB_URL=postgres://$(TEST_POSTGRES_USER):$(TEST_POSTGRES_PASSWORD)@$(TEST_POSTGRES_HOST):$(TEST_POSTGRES_PORT)/$(TEST_POSTGRES_DB)?sslmode=$(TEST_POSTGRES_SSL)

test-db-setup: test-db-up test-migrate-up
	@echo "Test database setup complete!"

test-db-teardown: test-db-down
	@echo "Test database teardown complete!"

test-e2e:
	@echo "Running E2E tests..."
	go test -v -tags=e2e ./tests -run TestE2E

benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -benchtime=10s ./tests -run=^$$ | tee benchmark_results.txt


test-db-up:
	@echo "Starting test database in Docker..."
	@docker-compose --env-file tests/.env.tests -f tests/docker-compose.test.yml up -d
	sleep 5
	@echo "Test database is ready!"

test-db-down:
	@echo "Stopping test database..."
	docker-compose -f tests/docker-compose.test.yml down -v
	@echo "Test database stopped!"

test-migrate-up:
	@echo "Running migrations up on test database..."
	goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) "$(TEST_DB_URL)" up