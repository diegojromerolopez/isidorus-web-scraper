# Nube2e

The main aim of this repository is to serve as a showcase of how to use localstack as a way to replace the AWS services an application is based on, and create e2e tests.

This application is a web scraper. Its main goal is to show the websites where a specific term appears.

## Use Cases

This application is designed for scenarios where deep content analysis of a web graph is required:

1.  **Brand Monitoring**: Find all mentions of a company name or product across a set of websites, including inside images (e.g., logos or product photos).
2.  **SEO & Backlink Analysis**: Map out the link structure between pages (stored in the adjacency list) and analyze term frequency on each node.
3.  **Compliance Auditing**: Crawl a network of sites to ensure specific terms (or prohibited terms) are present/absent.

## Project Structure

```text
.
├── api/                # FastAPI Application (Python)
├── workers/
│   ├── scraper/        # Recursive Web Scraper (Go)
│   ├── writer/         # Batch DB Writer (Go)
│   └── image_extractor/# Image Metadata Extractor (Python)
├── tests/
│   ├── unit/           # Python Unit Tests
│   └── e2e/            # End-to-End Test Suite
│       ├── runner/     # E2E Test Logic
│       └── mock_website/# Static target for scraping
├── Makefile            # Central Developer Entry Point
└── docker compose.*.yml# Infrastructure Orchestration
```

## Design Principles

The codebase follows several key design principles to ensure reliability and testability:

1.  **Domain-Driven Design (DDD)**: Logic is partitioned into:
    - **Domain**: Pure business models and logic.
    - **Services**: Orchestrates business processes, using repositories for side effects.
    - **Repositories/Clients**: Handles I/O operations (DB, SQS, Network).
2.  **Dependency Injection**: Services receive their dependencies (repositories/clients) as interfaces (Go) or objects (Python) during initialization.
3.  **Absolute Imports**: To ensure module resolution consistency across local development, Docker, and CI/CD, all Python imports are absolute (e.g., `from api.services...`).
4.  **Modern Python Typing**: Python 3.10+ syntax is required for all type hints:
    - Use `Type | None` instead of `Optional[Type]`.
    - Use lowercase `list`, `dict`, `tuple` instead of `List`, `Dict`, `Tuple`.
    - Avoid `Any` where possible; use `cast` only when necessary for library types.

## Architecture

The architecture consists of the following components:

### API (Python/FastAPI)
- **Routers**: Handle HTTP requests and use FastAPI's dependency injection to provide services.
- **Services**: Encapsulate business logic.
- **Mocks**: Integrated into the test suite to achieve **100% unit test coverage**.

### Workers
Workers are decoupled and highly testable through repository mocking.

1.  **Scrapers** (Golang):
    -   **Interface-based**: Uses `SQSClient`, `RedisClient`, and `PageFetcher` interfaces.
    -   **Logic**: Extracts terms, links, and image URLs. Supports recursive scraping via configurable depth.
    -   **Cycle Detection**: Uses Redis Sets (key: `scrape:{id}:visited`) to track handled URLs in a thread-safe, distributed manner.
    -   **Job Tracking**: Uses Redis for distributed reference counting to track pending tasks and signals job completion.

2.  **Image Extractor** (Python):
    -   Consumes image URLs found by the scraper.
    -   Extracts image metadata and links them to the source page and job.
    -   *Renamed from Image Explainer (AI removed for efficiency).*

3.  **Writer** (Golang):
    -   **Batch Processing**: Listens for results and performs efficient bulk inserts using a `DBRepository` interface.
    -   **Identifier Resolution**: Uses the provided internal integer ID for optimized storage and relationship mapping.
    -   **Job Management**: Updates job status in Postgres upon receiving completion signals.

## Configuration & Environment Variables

| Variable | Description | Default/Example |
|----------|-------------|-----------------|
| `AWS_ENDPOINT_URL` | LocalStack URL | `http://localstack:4566` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@localhost:5432/nube2e` |
| `INPUT_QUEUE_URL` | Queue for scrape requests | `http://localstack:4566/000000000000/scraper-input` |
| `WRITER_QUEUE_URL`| Queue for results to be written | `http://localstack:4566/000000000000/writer-queue` |
| `IMAGE_QUEUE_URL` | Queue for image processing | `http://localstack:4566/000000000000/image-queue` |

## Data Schema (PostgreSQL)

The system uses a denormalized schema optimized for fast text and graph queries. The `scrapings` table uses an internal Integer `id` for primary keys and a `uuid` for public identification.

- **`scrapings`**: Tracks scraping jobs. `id` (SERIAL PK).
- **`scraped_pages`**: Unique URLs and metadata. Linked via `scraping_id` (INT).
- **`page_terms`**: Map of terms and frequencies. Denormalized with `scraping_id` (INT) and `page_id` (INT).
- **`page_links`**: Adjacency list for the web graph. Denormalized with `scraping_id` (INT) and `source_page_id` (INT).
- **`page_images`**: Metadata and URLs. Denormalized with `scraping_id` (INT) and `page_id` (INT).

## Testing Strategy

### Unit Tests & Coverage
- **Purpose**: Verify business logic in isolation using mocks.
- **Coverage**: All core logic components targeted for **>90% coverage** (currently 100% for API/Scraper/Writer).
- **Execution**: Run via `make unit-test`.

### End-to-End (E2E) Tests
- **Infrastructure**: Uses `docker compose` with `docker-compose.e2e.yml` to spin up LocalStack and PostgreSQL.
- **Mock Website**: Decouples tests from the live internet.
- **Execution**: Run via `make test`.

## Development Guidelines

1.  **Test-Driven Development**: Always implement unit tests for new service logic.
2.  **Mocking Side Effects**: Do not make real network or DB calls in unit tests; use the repository interfaces.
3.  **Interface Consistency**: When updating Go repositories, update both the interface and the implementation to maintain testability.
4.  **Python Code Quality Standards**: All Python code must pass the following checks:
    - **Formatting**: `black` (88 char limit).
    - **Fast Linting**: `ruff` (used for imports sorting and general linting).
    - **Style/Bugs**: `flake8` (with Black-compatible config) and `pylint` (targeting a 10.0 score).
    - **Static Analysis**: `mypy` with strict mode (`disallow_untyped_defs = true`).
5.  **CI/CD Pipeline**: 
    - `tests-unit.yml`: Executes unit tests for all components.
    - `python-lint.yml`: Executes the full Python linting suite.
