# Makefile for Isidorus Web Scraper

.PHONY: up down build logs test clean unit-test lint lint-check format

# Default target
all: up

# Start all services in detached mode
up:
	docker compose -f docker-compose.base.yml -f docker-compose.yml up -d --build

# Stop all services and remove containers
down:
	docker compose -f docker-compose.base.yml -f docker-compose.yml down

# --- Production Environment ---

# Start production stack (Minio, ScyllaDB, ElasticMQ)
prod-up:
	docker compose -f docker-compose.base.yml -f docker-compose.prod.yml up -d --build

# Stop production stack
prod-down:
	docker compose -f docker-compose.base.yml -f docker-compose.prod.yml down

# Rebuild production stack
prod-build:
	docker compose -f docker-compose.base.yml -f docker-compose.prod.yml build

# ------------------------------

# Rebuild all services
build:
	docker compose -f docker-compose.base.yml -f docker-compose.yml build

# Follow logs from all services
logs:
	docker compose -f docker-compose.base.yml -f docker-compose.yml logs -f

# Run end-to-end tests
test-e2e:
	docker compose -f docker-compose.base.yml -f docker-compose.yml -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from test-runner

# Run basic end-to-end tests (no AI workers)
test-e2e-basic:
	IMAGE_EXPLAINER_ENABLED=false PAGE_SUMMARIZER_ENABLED=false \
	docker compose -f docker-compose.base.yml -f docker-compose.yml -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from test-runner \
		postgres localstack redis api auth-admin scraper-worker writer-worker indexer-worker opensearch deletion-worker mock-website test-runner

# Run the stack and trigger a scrape job
run: migrate seed-db
	SCRAPE_URL=$(URL) SCRAPE_DEPTH=$(DEPTH) docker compose -f docker-compose.base.yml -f docker-compose.yml -f docker-compose.run.yml up --build --abort-on-container-exit --exit-code-from trigger

# Run Django migrations and seed data
migrate:
	docker compose -f docker-compose.base.yml -f docker-compose.yml up -d postgres
	sleep 5
	docker compose -f docker-compose.base.yml -f docker-compose.yml run --rm auth-admin python manage.py migrate --fake-initial

seed-db:
	docker compose -f docker-compose.base.yml -f docker-compose.yml run --rm auth-admin python manage.py setup_test_data

# Run unit tests
test-unit:
	@echo "Running API unit tests..."
	docker build -t isidorus-api-test -f api/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app isidorus-api-test sh -c "pip install coverage && coverage run --branch --source=api -m unittest discover -v -s tests/unit/api -t /app && coverage report"
	@echo "Running Image Extractor unit tests..."
	docker run --rm -v "$$(pwd):/app" -w /app/workers/image_extractor golang:1.25-alpine sh -c "go mod tidy && go test -v -cover ./..."
	@echo "Running Image Explainer unit tests..."
	docker build -t isidorus-explainer-test -f workers/image_explainer/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app isidorus-explainer-test sh -c "pip install coverage && coverage run --branch --source=workers/image_explainer -m unittest discover -v -p 'test_*.py' -s tests/unit/workers/image_explainer -t /app && coverage report"
	@echo "Running Scraper unit tests..."
	docker run --rm -v "$$(pwd):/app" -w /app/workers/scraper golang:1.25-alpine sh -c "go mod tidy && go test -v -cover ./..."
	@echo "Running Writer unit tests..."
	docker run --rm -v "$$(pwd):/app" -w /app/workers/writer golang:1.25-alpine sh -c "go mod tidy && go test -v -cover ./..."
	@echo "Running Page Summarizer unit tests..."
	docker build -t isidorus-summarizer-test -f workers/page_summarizer/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app isidorus-summarizer-test sh -c "pip install coverage && coverage run --branch --source=workers/page_summarizer -m unittest discover -v -p 'test_*.py' -s tests/unit/workers/page_summarizer -t /app && coverage report"
	@echo "Running Deletion worker unit tests..."
	docker build -t isidorus-deletion-test -f workers/deletion/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app isidorus-deletion-test sh -c "pip install coverage && coverage run --branch --source=workers/deletion -m unittest discover -v -p 'test_*.py' -s tests/unit/workers/deletion -t /app && coverage report"
	@echo "Running Indexer unit tests..."
	docker run --rm -v "$$(pwd):/app" -w /app/workers/indexer golang:1.25-alpine sh -c "go mod tidy && go test -v -cover ./..."

# Clean up volumes and orphans
clean:
	docker compose -f docker-compose.yml -f docker-compose.e2e.yml -f docker-compose.run.yml down -v --remove-orphans

# Format code
format:
	@echo "Formatting Python code with black..."
	black .
	@echo "Sorting Python imports with isort..."
	isort .
	@echo "Sorting Python imports with ruff..."
	ruff check --select I --fix .
	@echo "Formatting Go code..."
	@docker run --rm -v "$$(pwd):/app" -w /app/workers/scraper golang:1.25-alpine go fmt ./...
	@docker run --rm -v "$$(pwd):/app" -w /app/workers/writer golang:1.25-alpine go fmt ./...
	@docker run --rm -v "$$(pwd):/app" -w /app/workers/indexer golang:1.25-alpine go fmt ./...
	@docker run --rm -v "$$(pwd):/app" -w /app/workers/image_extractor golang:1.25-alpine go fmt ./...
	@echo "Code formatting complete!"

# Check linting errors (does not modify files)
lint-check:
	@echo "Checking code formatting with black..."
	black --check .
	@echo "Checking import sorting with isort..."
	isort --check .
	@echo "Running ruff linter..."
	ruff check .
	@echo "Running mypy type checker..."
	mypy .
	@echo "Running pylint..."
	pylint --fail-under=9.5 api/ shared/ trigger/ workers/image_explainer/ workers/page_summarizer/ workers/deletion/ tests/unit/ tests/e2e/runner/runner.py
	@echo "Checking Go code formatting..."
	@if [ -n "$$(docker run --rm -v "$$(pwd):/app" -w /app/workers/scraper golang:1.25-alpine gofmt -l .)" ]; then \
		echo "Go formatting errors found in scraper worker. Run 'make format' to fix."; \
		docker run --rm -v "$$(pwd):/app" -w /app/workers/scraper golang:1.25-alpine gofmt -l .; \
		exit 1; \
	fi
	@if [ -n "$$(docker run --rm -v "$$(pwd):/app" -w /app/workers/writer golang:1.25-alpine gofmt -l .)" ]; then \
		echo "Go formatting errors found in writer worker. Run 'make format' to fix."; \
		docker run --rm -v "$$(pwd):/app" -w /app/workers/writer golang:1.25-alpine gofmt -l .; \
		exit 1; \
	fi
	@if [ -n "$$(docker run --rm -v "$$(pwd):/app" -w /app/workers/image_extractor golang:1.25-alpine gofmt -l .)" ]; then \
		echo "Go formatting errors found in image extractor worker. Run 'make format' to fix."; \
		docker run --rm -v "$$(pwd):/app" -w /app/workers/image_extractor golang:1.25-alpine gofmt -l .; \
		exit 1; \
	fi


# Run all linters and show errors (alias for lint-check)
lint: lint-check

# --- Kubernetes & Registry ---

DOCKER_USER ?= $(USER)
TAG ?= latest

dockerhub-push:
	@if [ -n "$$DOCKERHUB_TOKEN" ] && [ -n "$$DOCKERHUB_USERNAME" ]; then \
		./scripts/dockerhub-login.sh; \
	else \
		echo "Logging into Docker Hub (Interactive)..."; \
		docker login -u $(DOCKER_USER); \
	fi
	DOCKER_USER=$(DOCKER_USER) TAG=$(TAG) BRANCH=$(BRANCH) ./scripts/dockerhub-push.sh

k8s-secrets:
	./scripts/k8s-manage-secrets.sh apply

k8s-deploy: k8s-secrets
	kubectl apply -f k8s/base/namespace.yaml
	kubectl apply -f k8s/infra/
	kubectl apply -f k8s/apps/

k8s-update-images:
	DOCKER_USER=$(DOCKER_USER) TAG=$(TAG) ./scripts/k8s-update-images.sh

k8s-port-forward:
	./scripts/k8s-port-forward.sh

k8s-setup-all:
	./scripts/k8s-setup.sh

dockerhub-cleanup:
	./scripts/dockerhub-cleanup.sh

.PHONY: all up down build logs test test-e2e test-e2e-basic run migrate seed-db test-unit clean format lint-check lint k8s-secrets k8s-deploy dockerhub-push k8s-update-images k8s-port-forward k8s-setup-all dockerhub-cleanup
