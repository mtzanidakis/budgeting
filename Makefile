.PHONY: build run test clean cli docker-build docker-up docker-down

# Build the server and CLI
build:
	CGO_ENABLED=1 go build -o bin/server ./cmd/server
	CGO_ENABLED=1 go build -o bin/cli ./cmd/cli

# Run the server locally
run: build
	SESSION_SECRET=development-secret ./bin/server

# Run tests
test:
	go test -v ./...

# Run CLI
cli: build
	SESSION_SECRET=development-secret ./bin/cli $(ARGS)

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
	@$(MAKE) cli ARGS="user:add $(filter-out $@,$(MAKECMDGOALS))"

user-list:
	@$(MAKE) cli ARGS="user:list"

user-edit:
	@$(MAKE) cli ARGS="user:edit $(filter-out $@,$(MAKECMDGOALS))"

user-delete:
	@$(MAKE) cli ARGS="user:delete $(filter-out $@,$(MAKECMDGOALS))"

actions-query:
	@$(MAKE) cli ARGS="actions:query $(filter-out $@,$(MAKECMDGOALS))"

%:
	@:
