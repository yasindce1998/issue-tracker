# Environment variables
include .env
export

# Tool versions
GOLANGCI_LINT_VERSION := v1.60.1

# Docker-related variables
DOCKER_IMAGE_NAME := issue-tracker
DOCKER_IMAGE_TAG := latest
DOCKER_COMPOSE_FILE := docker-compose.yml
DOCKER_COMPOSE := docker compose -f $(DOCKER_COMPOSE_FILE)

# Proto-related variables
PROTO_DIR := pkg/pb
IMPORT_VALIDATE_PROTO_DIR := proto/validate
PROTO_IMPORT := -I. -I$(IMPORT_VALIDATE_PROTO_DIR) -I$(PROTO_DIR)

# Proto service definitions
USER_PROTO_FILE := user/v1/user.proto
USER_SERVICE := user.v1.UserService
ISSUES_PROTO_FILE := issues/v1/issues.proto
ISSUES_SERVICE := issues.v1.IssuesService
PROJECT_PROTO_FILE := project/v1/project.proto
PROJECT_SERVICE := project.v1.ProjectService

# Proto generation settings
GO_OUT := --go_out=. --go_opt=paths=source_relative
GRPC_OUT := --go-grpc_out=. --go-grpc_opt=paths=source_relative
VALIDATE_OUT := --validate_out="lang=go:."
GATEWAY_OUT := --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative
OPENAPI_OUT := --openapiv2_out=. --openapiv2_opt=logtostderr=true

# Example IDs for testing
ISSUE_ID := 09e86dfc-927f-43be-aab9-2cd64e1ecaa6
PROJECT_ID := 0cfc6cee-67a0-4b2a-8a99-4afa10d4a143
ASSIGNEE_ID := 623e4567-e89b-12d3-a456-426614174005
USER_ID := 623e4567-e89b-12d3-a456-426614174005

# Build targets
.PHONY: build build-all clean
build: ## Build the application
	go build -o bin/issue-tracker ./cmd

build-all: ## Build for multiple platforms (linux, darwin, windows)
	GOOS=linux GOARCH=amd64 go build -o bin/issue-tracker-linux-amd64 ./cmd
	GOOS=darwin GOARCH=amd64 go build -o bin/issue-tracker-darwin-amd64 ./cmd
	GOOS=windows GOARCH=amd64 go build -o bin/issue-tracker-windows-amd64.exe ./cmd

# Docker targets
.PHONY: docker-build docker-push docker-run
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) .

docker-push: ## Push Docker image to registry
	docker push $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

docker-run: ## Run application in Docker container
	docker run --rm -p 8080:8080 -p 9090:9090 --env-file .env $(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)

# Docker Compose targets
.PHONY: up down restart logs ps migrate-up migrate-down
up: ## Start all services with Docker Compose
	$(DOCKER_COMPOSE) up -d

down: ## Stop all services and remove containers
	$(DOCKER_COMPOSE) down

restart: ## Restart all services
	$(DOCKER_COMPOSE) restart

logs: ## Show logs for all services
	$(DOCKER_COMPOSE) logs -f

ps: ## List running containers
	$(DOCKER_COMPOSE) ps

migrate-up: ## Run database migrations
	$(DOCKER_COMPOSE) run --rm migrate up

migrate-down: ## Rollback database migrations
	$(DOCKER_COMPOSE) run --rm migrate down

# Database targets
.PHONY: db-start db-stop db-reset
db-start: ## Start database container only
	$(DOCKER_COMPOSE) up -d postgres

db-stop: ## Stop database container
	$(DOCKER_COMPOSE) stop postgres

db-reset: ## Reset database (caution: deletes all data)
	$(DOCKER_COMPOSE) down -v postgres
	$(DOCKER_COMPOSE) up -d postgres

clean: ## Clean build artifacts
	rm -rf bin/

# Code quality targets
.PHONY: lint vendor format imports tidy-code test cover
lint: ## Run go linter using golangci-lint in Docker
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run --timeout 2m

vendor: ## Update vendors -- run this before committing
	@go mod tidy
	@go mod vendor
	@git status vendor

format: ## Format Go code using gofmt
	@echo "Formatting code with gofmt..."
	@gofmt -w -s ./pkg ./cmd ./logger ./models ./database
	@echo "Done."

imports: ## Organize imports using goimports
	@echo "Installing goimports if needed..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "Organizing imports..."
	@goimports -w ./pkg ./cmd ./logger
	@echo "Done."

tidy-code: format imports ## Run both format and imports commands
	@echo "Code formatting complete!"

test: ## Run tests
	go test -v ./...

cover: ## Run tests with coverage and generate HTML report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Run target
.PHONY: run
run: ## Run the application
	go run ./cmd

# Proto generation targets
.PHONY: gen-user gen-issues gen-project
gen-user: ## Generate Go code and OpenAPI spec from user proto
	protoc $(PROTO_IMPORT) $(GO_OUT) $(GRPC_OUT) $(VALIDATE_OUT) $(GATEWAY_OUT) $(OPENAPI_OUT) $(PROTO_DIR)/$(USER_PROTO_FILE)

gen-issues: ## Generate Go code from issues proto
	protoc $(PROTO_IMPORT) $(GO_OUT) $(GRPC_OUT) $(VALIDATE_OUT) $(GATEWAY_OUT) $(OPENAPI_OUT) $(PROTO_DIR)/$(ISSUES_PROTO_FILE)

gen-project: ## Generate Go code from project proto
	protoc $(PROTO_IMPORT) $(GO_OUT) $(GRPC_OUT) $(VALIDATE_OUT) $(GATEWAY_OUT) $(OPENAPI_OUT) $(PROTO_DIR)/$(PROJECT_PROTO_FILE)

# User service gRPC calls
.PHONY: grpc-create-user grpc-get-user grpc-update-user grpc-delete-user grpc-list-users
grpc-create-user: ## Call CreateUser via grpcurl
	grpcurl -plaintext \
		-d '{"firstName":"John","lastName":"Ali","emailAddress":"john@example.com"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(USER_PROTO_FILE) \
		$(GRPC_SERVER) $(USER_SERVICE)/CreateUser

grpc-get-user: ## Call GetUser via grpcurl
	grpcurl -plaintext \
		-d '{"userId":"$(USER_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(USER_PROTO_FILE) \
		$(GRPC_SERVER) $(USER_SERVICE)/GetUser

grpc-update-user: ## Call UpdateUser via grpcurl
	grpcurl -plaintext \
		-d '{"userId":"$(USER_ID)","firstName":"Updated","lastName":"Name","emailAddress":"updated@example.com"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(USER_PROTO_FILE) \
		$(GRPC_SERVER) $(USER_SERVICE)/UpdateUser

grpc-delete-user: ## Call DeleteUser via grpcurl
	grpcurl -plaintext \
		-d '{"userId":"$(USER_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(USER_PROTO_FILE) \
		$(GRPC_SERVER) $(USER_SERVICE)/DeleteUser

grpc-list-users: ## Call ListUsers via grpcurl
	grpcurl -plaintext \
		-d '{"pageSize":10,"pageToken":""}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROTO_DIR)/$(USER_PROTO_FILE) \
		$(GRPC_SERVER) $(USER_SERVICE)/ListUsers

# Issues service gRPC calls
.PHONY: grpc-create-issue grpc-create-issue-no-assignee grpc-get-issue grpc-update-issue grpc-delete-issue grpc-list-issues
grpc-create-issue: ## Call CreateIssue via grpcurl
	grpcurl -plaintext \
		-d '{"summary":"New Feature","description":"Updated description","projectId":"$(PROJECT_ID)","assigneeId":"$(ASSIGNEE_ID)","priority":2,"type":1}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/CreateIssue

grpc-create-issue-no-assignee: ## Call CreateIssue via grpcurl
	grpcurl -plaintext \
		-d '{"summary":"New Feature","description":"Updated description","projectId":"$(PROJECT_ID)","priority":2,"type":1}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/CreateIssue

grpc-get-issue: ## Call GetIssue via grpcurl
	grpcurl -plaintext \
		-d '{"issueId":"$(ISSUE_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/GetIssue

grpc-update-issue: ## Call UpdateIssue via grpcurl
	grpcurl -plaintext \
		-d '{"issueId":"$(ISSUE_ID)","summary":"Updated Bug","description":"Updated description","assigneeId":"$(ASSIGNEE_ID)","priority":2,"status":3,"type":2}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/UpdateIssue

grpc-delete-issue: ## Call DeleteIssue via grpcurl
	grpcurl -plaintext \
		-d '{"issueId":"$(ISSUE_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/DeleteIssue

grpc-list-issues: ## Call ListIssues via grpcurl
	grpcurl -plaintext \
		-d '{"pageSize":10,"pageToken":""}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(ISSUES_PROTO_FILE) \
		$(GRPC_SERVER) $(ISSUES_SERVICE)/ListIssues

# Project service gRPC calls
.PHONY: grpc-create-project grpc-get-project grpc-update-project grpc-delete-project grpc-list-projects grpc-stream-projects
grpc-create-project: ## Call CreateProject via grpcurl
	grpcurl -plaintext \
		-d '{"name":"New Project","description":"Project description"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/CreateProject

grpc-get-project: ## Call GetProject via grpcurl
	grpcurl -plaintext \
		-d '{"projectId":"$(PROJECT_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/GetProject

grpc-update-project: ## Call UpdateProject via grpcurl
	grpcurl -plaintext \
		-d '{"projectId":"$(PROJECT_ID)","name":"Updated Project","description":"Updated description"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/UpdateProject

grpc-delete-project: ## Call DeleteProject via grpcurl
	grpcurl -plaintext \
		-d '{"projectId":"$(PROJECT_ID)"}' \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/DeleteProject

grpc-list-projects: ## Call ListProjects via grpcurl
	grpcurl -plaintext \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/ListProjects

grpc-stream-projects: ## Call StreamProjectUpdates via grpcurl (may not work, use Postman)
	grpcurl -plaintext \
		-import-path . \
		-import-path $(IMPORT_VALIDATE_PROTO_DIR) \
		-import-path $(PROTO_DIR) \
		-proto $(PROJECT_PROTO_FILE) \
		$(GRPC_SERVER) $(PROJECT_SERVICE)/StreamProjectUpdates