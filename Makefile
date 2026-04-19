.PHONY: build run test clean admin docker-build docker-up docker-down

# Version from git commit hash, fallback to "dev"
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/mtzanidakis/budgeting/internal/version.Version=$(VERSION)

# Build the server, admin CLI and budgeting-cli
build:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o bin/server ./cmd/server
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o bin/admin ./cmd/admin
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/budgeting-cli ./cmd/budgeting-cli

# Run the server locally
run: build
	SESSION_SECRET=development-secret ./bin/server

# Run tests
test:
	go test -v ./...

# Run admin CLI
admin: build
	SESSION_SECRET=development-secret ./bin/admin $(ARGS)

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f data/budgeting.db

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Development helpers
dev: clean run

# User management shortcuts
user-add:
	@$(MAKE) admin ARGS="user:add $(filter-out $@,$(MAKECMDGOALS))"

user-list:
	@$(MAKE) admin ARGS="user:list"

user-edit:
	@$(MAKE) admin ARGS="user:edit $(filter-out $@,$(MAKECMDGOALS))"

user-delete:
	@$(MAKE) admin ARGS="user:delete $(filter-out $@,$(MAKECMDGOALS))"

actions-query:
	@$(MAKE) admin ARGS="actions:query $(filter-out $@,$(MAKECMDGOALS))"

%:
	@:
