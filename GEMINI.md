# Isidorus Web Scraper

**About the Name**: Isidorus is named after **Isidore of Seville** (c. 560-636 AD), the renowned scholar and Archbishop of Seville who compiled the *Etymologiae*, the first encyclopedia of all human knowledge. Known as "The Schoolmaster of the Middle Ages," Isidore meticulously gathered, organized, and preserved the wisdom of his time across all fields of knowledge. Just as Isidore collected and systematized information, this application scrapes and archives web content.

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
│   ├── image_extractor/# Image Metadata Extractor (Python)
│   └── page_summarizer/# AI Page Summarizer (Python)
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
- **Services**: Encapsulate business logic with **async/await** for non-blocking I/O.
- **Async Clients**: Uses `redis.asyncio` for Redis operations, `aioboto3` for DynamoDB/SQS interactions, and `httpx` for HTTP requests.
- **Mocks**: Integrated into the test suite to achieve **100% unit test coverage**.

### Workers
Workers are decoupled and highly testable through repository mocking.

1.  **Scrapers** (Golang):
    -   **Interface-based**: Uses `SQSClient`, `RedisClient`, and `PageFetcher` interfaces.
    -   **Logic**: Extracts terms, links, and image URLs. Supports recursive scraping via configurable depth.
    -   **Cycle Detection**: Uses Redis Sets (key: `scrape:{id}:visited`) to track handled URLs in a thread-safe, distributed manner.
    -   **Job Tracking**: Uses Redis for distributed reference counting to track pending tasks and signals job completion.
    -   **Feature Flags**: Conditionally enables `IMAGE_EXPLAINER_ENABLED` and `PAGE_SUMMARIZER_ENABLED`.

2.  **Image Extractor** (Python):
    -   Consumes image URLs found by the scraper (queue: `image-extractor-queue`).
    -   Extracts image metadata and links them to the source page and job.
    -   *Renamed from Image Explainer (AI removed for efficiency).*

3.  **Page Summarizer** (Python):
    -   Consumes text content found by the scraper (queue: `page-summarizer-queue`).
    -   Generates concise summaries of web pages using LLMs.

4.  **Writer** (Golang):
    -   **Batch Processing**: Listens for results and performs efficient bulk inserts using a `DBRepository` interface.
    -   **Identifier Resolution**: Uses the provided internal integer ID for optimized storage and relationship mapping.
    -   **Job Management**: Updates job status in Postgres upon receiving completion signals.

## Configuration & Environment Variables

| Variable | Description | Default/Example |
|----------|-------------|-----------------|
| `AWS_ENDPOINT_URL` | LocalStack URL | `http://localstack:4566` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@localhost:5432/isidorus` |
| `INPUT_QUEUE_URL` | Queue for scrape requests | `http://localstack:4566/000000000000/scraper-input` |
| `WRITER_QUEUE_URL`| Queue for results to be written | `http://localstack:4566/000000000000/writer-queue` |
| `IMAGE_QUEUE_URL` | Queue for image processing | `http://localstack:4566/000000000000/image-extractor-queue` |
| `SUMMARIZER_QUEUE_URL`| Queue for page summarizer | `http://localstack:4566/000000000000/page-summarizer-queue` |
| `DYNAMODB_TABLE` | DynamoDB Table Name | `scraping_jobs` |
| `IMAGE_EXPLAINER_ENABLED` | Enable AI image explanation | `true` |
| `PAGE_SUMMARIZER_ENABLED` | Enable page summarization | `true` |

## Data Schema (PostgreSQL & DynamoDB)

The system uses a hybrid schema:
- **DynamoDB**: Key-value store for Job History.
- **PostgreSQL**: Relational graph data for scraped content.

### DynamoDB
- **`scraping_jobs`**: Logs job lifecycle.
  - PK: `job_id` (String)
  - Attributes: `url`, `depth`, `status`

### PostgreSQL
The `scrapings` table uses an internal Integer `id` for primary keys and a `uuid` for public identification.

- **`scrapings`**: Tracks scraping jobs. `id` (SERIAL PK).
- **`scraped_pages`**: Unique URLs and metadata. Linked via `scraping_id` (INT).
- **`page_terms`**: Map of terms and frequencies. Denormalized with `scraping_id` (INT) and `page_id` (INT).
- **`page_links`**: Adjacency list for the web graph. Denormalized with `scraping_id` (INT) and `source_page_id` (INT).
- **`page_images`**: Metadata and URLs. Denormalized with `scraping_id` (INT) and `page_id` (INT).

## Testing Strategy

### Unit Tests & Coverage
- **Purpose**: Verify business logic in isolation using mocks.
- **Coverage**: All core logic components targeted for **100% coverage**.
- **Execution**: Run via `make test-unit`.

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
6.  **Virtual Environment**: Use a Python virtual environment to isolate project dependencies. Do not install packages in the system Python runtime.
7.  **Async I/O Operations**: All Python I/O operations must be asynchronous to prevent blocking the event loop:
    - **Database**: Use Tortoise ORM (already async).
    - **Redis**: Use `redis.asyncio`.
    - **AWS Services**: Use `aioboto3` for SQS, DynamoDB, and S3.
    - **HTTP Requests**: Use `httpx.AsyncClient` instead of `requests`.
    - **File I/O**: Use `aiofiles` if needed (currently not used).
    - Never use synchronous libraries like `requests`, `boto3` (use `aioboto3`), or `redis` (use `redis.asyncio`) in async contexts.
8.  **Pre-Commit Linting and Testing**: All code changes must pass linting checks and tests before committing:
    - Run `make format` to auto-format code with `black` and sort imports with `ruff`.
    - Run `make lint` to verify all linting checks pass (`black`, `ruff`, `flake8`, `mypy`, `pylint`).
    - Run `make test-unit` to verify all unit tests pass.
    - Run `make test-e2e` to verify all end-to-end tests pass.
    - Target a PyLint score of **≥9.5/10**.
    - **All tests must pass before committing any changes.**
9.  **Private Methods and Attributes**: Use double underscore prefix (`__`) for truly private methods and attributes.
    - **Public interface**: Only expose methods and attributes that are part of the class's contract.
    - **Testing Private Members**: **Do not access private attributes or methods in tests** (e.g., `client._Class__attribute`). Instead, use `unittest.mock.patch` to mock dependencies or inject mocks via the constructor. Tests should verify behavior through the public interface.
