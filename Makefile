# Makefile for Nube2e

.PHONY: up down build logs test clean unit-test lint lint-check format

# Default target
all: up

# Start all services in detached mode
up:
	docker compose -f docker-compose.e2e.yml up -d --build

# Stop all services and remove containers
down:
	docker compose -f docker-compose.e2e.yml down

# Rebuild all services
build:
	docker compose -f docker-compose.e2e.yml build

# Follow logs from all services
logs:
	docker compose -f docker-compose.e2e.yml logs -f

# Run end-to-end tests
test-e2e:
	docker compose -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from test-runner

# Run unit tests
test-unit:
	@echo "Running API unit tests..."
	docker build -t nube2e-api-test -f api/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app nube2e-api-test sh -c "pip install coverage && coverage run --branch --source=api -m unittest discover -v -s tests/unit/api -t /app && coverage report"
	@echo "Running Image Extractor unit tests..."
	docker build -t nube2e-extractor-test -f workers/image_extractor/Dockerfile .
	docker run --rm -v "$$(pwd):/app" -e PYTHONPATH=/app nube2e-extractor-test sh -c "pip install coverage && coverage run --branch --source=workers/image_extractor -m unittest discover -v -p 'test_*.py' -s tests/unit/workers/image_extractor -t /app && coverage report"
	@echo "Running Scraper unit tests..."
	docker build -t nube2e-scraper-test workers/scraper/
	docker run --rm -v "$$(pwd):/app" -w /app/workers/scraper nube2e-scraper-test sh -c "go mod tidy && go test -v -cover ./..."
	@echo "Running Writer unit tests..."
	docker build -t nube2e-writer-test workers/writer/
	docker run --rm -v "$$(pwd):/app" -w /app/workers/writer nube2e-writer-test sh -c "go mod tidy && go test -v -cover ./..."

# Clean up volumes and orphans
clean:
	docker compose -f docker-compose.e2e.yml down -v --remove-orphans

# Format Python code with black
format:
	@echo "Formatting Python code with black..."
	black .
	@echo "Sorting imports with ruff..."
	ruff check --select I --fix .
	@echo "Code formatting complete!"

# Check linting errors (does not modify files)
lint-check:
	@echo "Checking code formatting with black..."
	black --check .
	@echo "Running ruff linter..."
	ruff check .
	@echo "Running flake8..."
	flake8 .
	@echo "Running mypy type checker..."
	mypy .
	@echo "Running pylint..."
	pylint api/ workers/image_extractor/ tests/unit/ tests/e2e/runner/runner.py
	@echo "All linting checks passed!"

# Run all linters and show errors (alias for lint-check)
lint: lint-check

