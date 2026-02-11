# Isidorus Web Scraper

**About the Name**: Isidorus is named after **Isidore of Seville** (c. 560-636 AD), the renowned scholar and Archbishop of Seville who compiled the *Etymologiae*, the first encyclopedia of all human knowledge. Known as "The Schoolmaster of the Middle Ages," Isidore meticulously gathered, organized, and preserved the wisdom of his time across all fields of knowledge. Just as Isidore collected and systematized information, this application scrapes and archives web content.

The main aim of this repository is to serve as a showcase of how to use localstack as a way to replace the AWS services an application is based on, and create e2e tests. 

Furthermore, it demonstrates a **provider-agnostic architecture**, showing how an application designed for AWS can be seamlessly transitioned to self-hosted, open-source alternatives like Minio, ElasticMQ, and ScyllaDB.

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
├── auth_admin/         # API Key & User Management (Django)
├── workers/
│   ├── scraper/        # Recursive Web Scraper (Go)
│   ├── writer/         # Batch DB Writer (Go)
│   ├── image_extractor/# Image Metadata Extractor (Go)
│   ├── image_explainer/# AI Image Explainer (Python)
│   └── page_summarizer/# AI Page Summarizer (Python)
├── tests/
│   ├── unit/           # Python, Go, and Frontend Unit Tests
│   └── e2e/            # End-to-End Test Suite
│       ├── runner/     # E2E Test Logic (Python)
│       └── mock_website/# Static target for scraping
├── Makefile            # Central Developer Entry Point
└── docker compose.*.yml# Infrastructure Orchestration
```

## 🏗️ Showcases

### LocalStack Showcase (Development & E2E)
Emulates S3, SQS, and DynamoDB for local development and high-fidelity integration testing. This architecture is defined in `docker-compose.e2e.yml`.

```mermaid
graph TD
    subgraph "Development (LocalStack)"
        LS[LocalStack]
        S3_L[S3 Storage]
        SQS_L[SQS Queues]
        DDB_L[DynamoDB Table]
        LS --- S3_L
        LS --- SQS_L
        LS --- DDB_L
    end

    TR[Test Runner] -->|Monitor| API[API]
    API -->|Enqueue| SQS_L
    Scraper[Scraper Worker] -->|Consume| SQS_L
    Scraper -->|Scan| Site[Mock Website]
```

### Agnostic Infrastructure Showcase (Production)
Replaces cloud service dependencies with self-hosted, open-source alternatives. This architecture is defined in `docker-compose.prod.yml`.

```mermaid
graph TD
    subgraph "Production (Self-Hosted)"
        Minio[Minio - S3]
        EMQ[ElasticMQ - SQS]
        Scylla[ScyllaDB - DynamoDB]
    end

    API[API Service] -->|S3_URL| Minio
    API -->|SQS_URL| EMQ
    API -->|DYNAMO_URL| Scylla
    
    Workers[Worker Pool] --> Minio
    Workers --> EMQ
    Workers --> Scylla
```

### Kubernetes Showcase (Kind & KEDA)
Automates the deployment of the entire stack into a local [Kind](https://kind.sigs.k8s.io/) cluster and implements **Event-Driven Autoscaling** via [KEDA](https://keda.sh/).

```mermaid
graph TD
    subgraph "Kubernetes (Kind)"
        KEDA[KEDA Operator]
        SQS[SQS Queues]
        Workers[Worker Pods]
        HPA[Horizontal Pod Autoscaler]
        
        KEDA -->|Monitor| SQS
        KEDA -->|Trigger| HPA
        HPA -->|Scale| Workers
    end

    API[API Service] -->|Enqueue| SQS
    Workers -->|Consume| SQS
```

## Design Principles

The codebase follows several key design principles to ensure reliability and testability:

1.  **Domain-Driven Design (DDD)**: Logic is partitioned into:
    - **Domain**: Pure business models and logic.
    - **Services**: Orchestrates business processes, using repositories for side effects.
    - **Repositories/Clients**: Handles I/O operations (DB, SQS, Network).
2.  **Dependency Injection**: Services receive their dependencies (repositories/clients) as interfaces (Go) or objects (Python) during initialization.
3.  **Absolute Imports**: To ensure module resolution consistency across local development, Docker, and CI/CD, all Python imports are absolute (e.g., `from api.services...`).
4.  **Modern Python Typing**: Python 3.14+ syntax is required for all type hints:
    - Use `Type | None` instead of `Optional[Type]`.
    - Use lowercase `list`, `dict`, `tuple` instead of `List`, `Dict`, `Tuple`.
    - Avoid `Any` where possible; use `cast` only when necessary for library types.
5.  **Bleeding Edge Runtimes**: Always use the latest stable versions of runtimes to ensure performance and security:
    -   **Python**: Use the latest available version (currently 3.14).
        -   **Exception**: Workers using `langchain` (e.g., `page_summarizer`, `image_extractor`) must use **Python 3.11** as `langchain` dependencies (specifically `numpy` and `pandas`) do not yet support Python 3.14.
    -   **Go**: Use the latest available version (currently 1.25).

## Architecture

The architecture consists of the following components:

### API (Python/FastAPI)
- **Routers**: Handle HTTP requests and use FastAPI's dependency injection to provide services.
- **Services**: Encapsulate business logic with **async/await** for non-blocking I/O.
- **Async Clients**: Uses `redis.asyncio` for Redis operations, `aioboto3` for DynamoDB/SQS interactions, and `httpx` for HTTP requests.
- **Mocks**: Integrated into the test suite to achieve **100% unit test coverage**.

### Auth Admin (Python/Django)
- **Role**: Contol Plane for managing system state that isn't high-throughput.
- **Admin**: Uses the standard Django Admin interface for secure management of Users and API Keys.
- **Security**: Handles key generation and irreversible SHA-256 hashing.
- **Models**:
  - `APIKey`: Stores hashed keys, prefixes, owner relationship, and expiration.
  - `User`: Standard Django user model for authentication.
- **High Performance Validation**: While keys are managed in Django, they are stored in the shared `api_keys` table. The FastAPI service validates these keys using direct Postgres queries (via Tortoise ORM) and leverages **Redis** for sub-millisecond caching of valid keys.
- **Management Commands**:
  - `setup_test_data`: Seeds a known test key (`test-api-key-123`) for E2E tests.

### Workers
Workers are decoupled and highly testable through repository mocking.

1.  **Scrapers** (Golang):
    -   **Interface-based**: Uses `SQSClient`, `RedisClient`, and `PageFetcher` interfaces.
    -   **Logic**: Extracts terms, links, and image URLs. Supports recursive scraping via configurable depth.
    -   **Cycle Detection**: Uses Redis Sets (key: `scrape:{id}:visited`) to track handled URLs in a thread-safe, distributed manner.
    -   **Job Tracking**: Uses Redis for distributed reference counting to track pending tasks and signals job completion.
    -   **Feature Flags**: Conditionally enables `IMAGE_EXPLAINER_ENABLED` and `PAGE_SUMMARIZER_ENABLED`.

2.  **Image Extractor** (Golang):
    -   Consumes image URLs found by the scraper (queue: `image-extractor-queue`).
    -   **Persistence**: Downloads and uploads images to an AWS S3 bucket.
    -   Sends metadata to the Writer and enqueues tasks for the Explainer.

3.  **Image Explainer** (Python):
    -   Consumes tasks from `image-explainer-queue`.
    -   **S3 Retrieval**: Downloads images from AWS S3 for processing.
    -   **AI Description**: Generates image descriptions (alt-text) using LLM providers.
    -   Sends results to the Writer.

4.  **Indexer Worker** (Golang):
    -   Consumes text content and summaries (queue: `indexer-queue`).
    -   Indexes documents into **OpenSearch** for full-text search and relevance scoring.
    -   Replaces the legacy `page_terms` SQL storage with high-performance search capabilities.

4.  **Writer** (Golang):
    -   **Identifier Resolution**: Uses the provided internal integer ID for optimized storage and relationship mapping.
    -   **Job Management**: Updates job status in Postgres upon receiving completion signals.

5.  **Deletion Worker** (Python):
    -   **Async Processing**: Consumes deletion requests from `deletion-queue`.
    -   **Resource Cleanup**: Orchestrates the removal of S3 objects, PostgreSQL records, DynamoDB items, and OpenSearch documents.
    -   **Batching**: Uses batch operations to handle large scraping jobs efficiently.

## Configuration & Environment Variables

| Variable | Description | Default/Example |
|----------|-------------|-----------------|
| `BASE_ENDPOINT_URL` | Base service endpoint fallback | `http://localstack:4566` |
| `S3_ENDPOINT_URL` | Specific S3 endpoint override | `http://minio:9000` |
| `SQS_ENDPOINT_URL` | Specific SQS endpoint override | `http://elasticmq:9324` |
| `DYNAMODB_ENDPOINT_URL`| Specific DynamoDB endpoint override | `http://scylla:8042` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@localhost:5432/isidorus` |
| `INPUT_QUEUE_URL` | Queue for scrape requests | `http://localstack:4566/000000000000/scraper-input` |
| `WRITER_QUEUE_URL`| Queue for results to be written | `http://localstack:4566/000000000000/writer-queue` |
| `INDEXER_QUEUE_URL`| Queue for OpenSearch indexing | `http://localstack:4566/000000000000/indexer-queue` |
| `IMAGE_QUEUE_URL` | Queue for image processing | `http://localstack:4566/000000000000/image-extractor-queue` |
| `IMAGE_EXPL_QUEUE_URL`| Queue for image explanation | `http://localstack:4566/000000000000/image-explainer-queue` |
| `SUMMARIZER_QUEUE_URL`| Queue for page summarizer | `http://localstack:4566/000000000000/page-summarizer-queue` |
| `DYNAMODB_TABLE` | DynamoDB Table Name | `scraping_jobs` |
| `IMAGE_EXPLAINER_ENABLED` | Enable AI image explanation | `true` |
| `PAGE_SUMMARIZER_ENABLED` | Enable page summarization | `true` |
| `LLM_PROVIDER` | LLM backend (mock, openai, ollama) | `ollama` |

### Local LLM Support (Ollama)
The system supports running LLM workloads locally via **Ollama**. This is the default configuration in `docker-compose.yml`, using the `phi3` model for efficiency. Local inference ensures data privacy and eliminates external API costs.

## Data Schema (PostgreSQL & DynamoDB)

The system uses a triple-store architecture:
- **DynamoDB**: Key-value store for Job History.
- **OpenSearch**: Full-text index for scraped content and AI summaries.
- **PostgreSQL**: Relational graph data for scraped pages and links.

### DynamoDB
- **`scraping_jobs`**: Logs job lifecycle.
  - PK: `scraping_id` (String)
  - Attributes: `url`, `depth`, `status`

### PostgreSQL
The `scrapings` table uses an internal Integer `id` for primary keys and a `uuid` for public identification.

- **`scrapings`**: Tracks scraping jobs. `id` (SERIAL PK).
- **`scraped_pages`**: Unique URLs and metadata. Linked via `scraping_id` (INT).
- **`page_links`**: Adjacency list for the web graph. Denormalized with `scraping_id` (INT) and `source_page_id` (INT).
- **`page_images`**: Metadata and URLs. Denormalized with `scraping_id` (INT) and `page_id` (INT).
- **`api_keys`**: Managed by Django. Used for FastAPI authentication.

## Testing Strategy

### Unit Tests & Coverage
- **Purpose**: Verify business logic in isolation using mocks.
- **Coverage**: All core logic components (Backend & Frontend) targeted for **100% coverage**.
- **Frontend**: Tested using **Vitest** in `frontend/src/test/`.
- **Execution**: Run via `make test-unit`.

### End-to-End (E2E) Tests
- **Infrastructure**: Uses `docker compose` with `docker-compose.e2e.yml` to spin up LocalStack and PostgreSQL.
- **Mock Website**: Decouples tests from the live internet.
- **Execution**: Run via `make test-e2e`.

## ☸️ Running on Kubernetes (Kind)

The project provides high-fidelity local Kubernetes automation to mirror production environments.

### Automated Setup
The entire cluster lifecycle is managed via a single script:
```bash
make k8s-setup-all
```
This handles:
1.  **Cluster Creation**: Provisioning a Kind cluster.
2.  **KEDA Installation**: Deploying the event-driven autoscaling operator.
3.  **Image Loading**: Building and injecting all 10 services into the cluster nodes.
4.  **Manifest Application**: Deploying infrastructure (Postgres, Redis, ScyllaDB, OpenSearch, Ollama) and application workers.

### Event-Driven Autoscaling
The system utilizes [KEDA](https://keda.sh/) to scale workers based on **SQS queue depth** rather than CPU/Memory metrics. This ensures the system stays responsive during large crawl jobs and saves resources by scaling idle workers to zero.

| Component | Scaling Range | Metric |
|-----------|---------------|--------|
| **Scraper** | 1 - 10 | 5 messages / pod |
| **Image Extractor** | 0 - 10 | 5 messages / pod |
| **Ollama (Inference)** | 1 - 4 | Aggregated AI load |
| **AI Workers** | 0 - 8 | 1 message / pod |
| **Writer / Indexer**| 1 - 5 | 10 messages / pod |

## Development Guidelines

1.  **Test-Driven Development**: Always implement unit tests for new service logic.
2.  **Mocking Side Effects**: Do not make real network or DB calls in unit tests; use the repository interfaces.
3.  **Interface Consistency**: When updating Go repositories, update both the interface and the implementation to maintain testability.
4.  **Python Code Quality Standards**: All Python code must pass the following checks:
    - **Formatting**: `black` (88 char limit).
    - **Import Sorting**: `isort` (compatible with Black).
    - **Fast Linting**: `ruff` (used for general linting).
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
    - Run `make format` to auto-format code with `black` and search imports with `isort`.
    - Run `make lint` to verify all linting checks pass (`black`, `isort`, `ruff`, `flake8`, `mypy`, `pylint`).
    - Run `make test-unit` to verify all unit tests pass.
    - Run `make test-e2e` to verify all end-to-end tests pass.
    - Target a PyLint score of **≥9.5/10**.
    - **All tests must pass before committing any changes.**
9.  **Private Methods and Attributes**: Use double underscore prefix (`__`) for truly private methods and attributes. This includes all class instance attributes initialized in the `__init__` method, unless they are intended to be part of the public interface.
    - **Public interface**: Only expose methods and attributes that are part of the class's contract.
    - **Testing Private Members**: **Do not access private attributes or methods in tests** (e.g., `client._Class__attribute`). Instead, use `unittest.mock.patch` to mock dependencies or inject mocks via the constructor. Tests should verify behavior through the public interface.
10. **Constant-Driven Defaults**: Never define default values in function parameters with raw values (literals). Always use class constants or module-level constants to ensure maintainability and a single source of truth for configuration values.

## Docker Best Practices

1.  **Explicit Copying**: Never use `COPY . .` in Dockerfiles.
    -   **Why**: It creates a large build context, invalidates the cache on any file change (even irrelevant ones like `README.md`), and risks including sensitive files.
    -   **Rule**: Always copy only the specific files and directories needed for the build (e.g., `COPY main.go .`, `COPY config/ config/`).
2.  **Multi-Stage Builds**: Use multi-stage builds to keep production images small (e.g., building Go binaries in a `builder` stage and copying only the binary to a scratch or alpine final image).
3.  **Rootless Containers**: Where possible, configure containers to run as non-root users for security.
