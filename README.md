<div align="center">
  <img src="docs/images/logo.png" alt="Isidorus Web Scraper Logo" width="300"/>
</div>

# Isidorus Web Scraper

[![Unit Tests](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/tests-unit.yml/badge.svg)](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/tests-unit.yml)
[![Python Lint](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/python-lint.yml/badge.svg)](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/python-lint.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.10+](https://img.shields.io/badge/python-3.10+-blue.svg)](https://www.python.org/downloads/)
[![Go 1.21+](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/dl/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.100+-009688.svg)](https://fastapi.tiangolo.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED.svg)](https://www.docker.com/)
[![LocalStack](https://img.shields.io/badge/LocalStack-AWS-4D4D4D.svg)](https://localstack.cloud/)
[![Code style: black](https://img.shields.io/badge/code%20style-black-000000.svg)](https://github.com/psf/black)
[![Ruff](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/astral-sh/ruff/main/assets/badge/v2.json)](https://github.com/astral-sh/ruff)
[![Imports: isort](https://img.shields.io/badge/%20imports-isort-%231674b1?style=flat&labelColor=ef8336)](https://pycqa.github.io/isort/)

**Isidorus Web Scraper** is named after **Isidore of Seville** (c. 560-636 AD), the renowned scholar and Archbishop of Seville who compiled the *Etymologiae*, the first encyclopedia of all human knowledge. Known as "The Schoolmaster of the Middle Ages," Isidore meticulously gathered, organized, and preserved the wisdom of his time. Just as Isidore collected and systematized knowledge, Isidorus Web Scraper archives and indexes web content with precision and thoroughness.

This project is a web scraping application designed to scrape, analyze, and archive web content. It demonstrates a robust, event-driven architecture using LocalStack to emulate AWS services.

**DISCLAIMER: this project was created by making use of the agentic AI models Gemini 3.0 Pro/Gemini 3.0 Flash/Claude Sonnet 4.5.**

## Overview

The application takes a URL and a search term, recursively scrapes the website to a configurable depth, and stores:
- **Pages**: Metadata of visited URLs.
- **Terms**: Frequency of the search term on each page.
- **Links**: graph of internal and external links.
- **Images**: Extracted image URLs, stored in S3 and explained by AI.

## Architecture

```mermaid
graph TD
    User((User)) -->|HTTP POST /scrape| API[API-FastAPI]
    API -->|1. Create Scraping| DB[(PostgreSQL)]
    API -->|2. Log Job Meta| Dynamo[(DynamoDB)]
    API -->|3. Start Job| SQS_S[SQS-Scraper Queue]
    
    Admin((Admin)) -->|Manage Keys| AuthAdmin[Auth Admin-Django]
    AuthAdmin -->|4. Sync Keys| DB
    API -->|5. Validate Key| Redis[(Redis)]
    API -->|6. Check DB| DB
    
    SQS_S --> Scraper[Scraper-Go]
    Scraper -->|7. Cycle Detection| Redis
    Scraper -->|8. Track Depth| Redis
    Scraper -->|9. Found Image| SQS_I[SQS-Image Queue]
    Scraper -->|10. Page Data| SQS_W[SQS-Writer Queue]
    
    SQS_I --> Extractor[Image Extractor-Python]
    Extractor -->|11. Upload Image| S3[(S3-LocalStack)]
    Extractor -->|12. Explain via LangChain| LLM((AI Models))
    Extractor -->|13. Explanation Result| SQS_W

    Scraper -->|10b. Page Text| SQS_PS[SQS-Summarizer Queue]
    SQS_PS --> Summarizer[Page Summarizer-Python]
    Summarizer -->|14. Summarize| LLM
    Summarizer -->|15. Summary Result| SQS_W
    
    SQS_W --> Writer[Writer-Go]
    Writer -->|16. Store Results| DB
    Writer -->|17. Mark Completion| Dynamo
```

## Data Storage Strategy

Isidorus uses a hybrid storage approach to optimize for both relational integrity and high-throughput status monitoring:

### 1. PostgreSQL (Relational Site Content)
**Location**: `scrapings`, `scraped_pages`, `page_terms`, `page_links`, `page_images`.
**Purpose**: Stores the core "knowledge graph" extracted from the web. 
**Why**: 
- **Relational Integrity**: Perfect for the complex relationships between pages, images, and search terms.
- **Query Flexibility**: Allows for complex joins and text searches.
- **Identity**: Acts as the system's "Identity Store" by generating unique incremental IDs for jobs.

### 2. DynamoDB (Job Lifecycle & State)
**Location**: `scraping_jobs` table.
**Purpose**: Stores the current **Status** (`PENDING`, `COMPLETED`), `created_at`, `completed_at`, and job-level metadata (`url`, `depth`).
**Why**:
- **Scaling Status Polling**: Offloads high-frequency status checks from the relational database.
- **NoSQL Flexibility**: Allows for job metadata that might vary across different scraping strategies.
- **Separation of Concerns**: Decouples the transient orchestration state (DynamoDB) from the permanent archived content (PostgreSQL).

### 3. Redis (Distributed Coordination)
**Location**: In-memory sets and counters.
**Purpose**: handles **Cycle Detection** and **Distributed Reference Counting** for job completion tracking in a multi-worker environment.

The system is built with a microservices approach:

0.  **Auth Admin (Django)**:
    -   Control plane for managing API keys and users.
    -   Securely hashes keys and provides a UI for revocation and expiration.
    -   Shares the PostgreSQL database with the API for high-performance validation.

1.  **API (FastAPI)**:
    -   Entry point for users.
    -   Enforces **API Key Authentication** with Redis caching for sub-millisecond validation.
    -   Initiates scraping jobs by sending messages to SQS.
    -   Tracks job status and results using Postgres and Redis.
    -   Provides endpoints to query results.
    -   Uses shared Python library for AWS clients and configuration.

2.  **Scraper Worker (Go)**:
    -   Consumes scrape requests from SQS.
    -   Fetches and parses HTML.
    -   Extracts terms, links, and images.
    -   **Cycle Prevention**: Uses Redis Sets (`SADD`) to track and skip already-processed URLs per scraping session.
    -   **Distributed Tracking**: Uses Redis for distributed reference counting to track job completion.
    -   Recursively enqueues links for further scraping.
    -   Conditionally sends data to Image Extractor and Page Summarizer based on feature flags.

3.  **Image Extractor Worker (Python)**:
    -   Consumes image URLs from `image-extractor-queue`.
    -   **S3 Persistence**: Downloads and uploads images to an AWS S3 bucket.
    -   **AI Explainer**: Generates image descriptions using LLM providers.
    -   Sends results to the Writer.

4.  **Page Summarizer Worker (Python)**:
    -   Consumes text content from `page-summarizer-queue`.
    -   **AI Summarization**: Generates concise summaries of web pages using LLMs.
    -   Sends results to the Writer.

5.  **Writer Worker (Go)**:
    -   Consumes structured data (pages, terms, links, images, job completion events) from SQS.
    -   Writes data to PostgreSQL in a normalized schema.
    -   Handles job completion status updates.

5.  **Shared Library (Python)**:
    -   Common package (`shared/`) containing reusable components.
    -   **Async AWS Clients**: `SQSClient` and `S3Client` using `aioboto3` for non-blocking I/O.
    -   **Configuration**: Base `Configuration` class for centralized environment variable management.
    -   Used by both API and Image Extractor worker to ensure consistency.

## Configuration & Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AWS_ENDPOINT_URL` | LocalStack URL | `http://localstack:4566` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@host:5432/db` |
| `REDIS_HOST` | Redis host | `localhost` or `redis` |
| `IMAGE_BUCKET` | S3 bucket for images | `isidorus-images` |
| `LLM_PROVIDER` | AI provider for explanations | `mock`, `openai`, `gemini`, etc. |
| `MAX_DEPTH` | Maximum recursive depth | `2` (Default from API) |
| `IMAGE_EXPLAINER_ENABLED` | Enable AI image explanation | `true` |
| `PAGE_SUMMARIZER_ENABLED` | Enable page summarization | `true` |

## API Endpoints

-   **`POST /scrape`**: Start a new scraping job.
    -   Body: `{"url": "...", "term": "...", "depth": 2}`
    -   **Example**:
        ```bash
        curl -X POST http://localhost:8000/scrape \
          -H "Content-Type: application/json" \
          -H "X-API-Key: test-api-key-123" \
          -d '{"url": "https://example.com", "term": "example", "depth": 1}'
          -d '{"url": "https://example.com", "term": "example", "depth": 1}'
        ```
        Response:
        ```json
        {"scraping_id": 123}
        ```
-   **`GET /scrape?scraping_id=1`**: Check status and get results of a scraping job.
-   **`GET /search?term={term}`**: Search for pages containing a specific term.
-   **`GET /terms`**: List all unique terms found across all scrapings.

## Authentication

The API requires an API Key for all requests. The key MUST be provided in the `X-API-Key` header.

### Creating an API Key

1.  **Access the Admin Interface**: Go to `http://localhost:8001/admin` (if running via Docker).
2.  **Login**: Use your superuser credentials.
3.  **Navigate to API Keys**: Click on "API Keys" under the "Authentication" section.
4.  **Add API Key**:
    -   Click "Add API Key".
    -   Select a user.
    -   Provide a descriptive name.
    -   (Optional) Set an expiration date.
5.  **Copy the Key**: Once you click Save, the **raw API Key will be displayed only once**. Copy it and store it securely.

### Developer Setup (Initial Key)

If you are running the environment for the first time, you can seed a default test key:
```bash
make migrate
make seed-db
```
This will create a key `test-api-key-123` for the user `test-runner`.

## Infrastructure

The entire stack runs locally via Docker Compose:
-   **LocalStack**: Emulates SQS and S3.
-   **PostgreSQL**: Relational database for scraping results and image metadata.
-   **DynamoDB**: NoSQL store for job history and metadata.
-   **Redis**: In-memory store for cycle detection and job tracking counters.

## Technologies

-   **Backend**: Python 3.10+ (FastAPI), Go 1.21+
-   **Async I/O**: `redis.asyncio` (async Redis client), `aioboto3` (async AWS SDK), `httpx` (async HTTP client)
-   **ORM**: Tortoise ORM (API)
-   **Database**: PostgreSQL 15
-   **Caching/Coordination**: Redis 7
-   **Infrastructure**: LocalStack (AWS SQS/S3/DynamoDB emulation), Docker Compose
-   **NoSQL**: DynamoDB (Job History)
-   **AI**: LangChain (Multi-provider support)
-   **Testing**: `unittest` (Python), `go test` (Go), `boto3`/`requests` (E2E)
-   **Mock Website**: A static site container for safe, deterministic E2E testing.

## Prerequisites

-   Docker & Docker Compose (v2+)
-   Python 3.10+
-   Go 1.21+
-   Make

## Getting Started

1.  **Start the environment**:
    ```bash
    make up
    ```

2.  **Run Full End-to-End Tests**:
    Includes image extraction and page summarization (requires more resources).
    ```bash
    make test-e2e
    ```

3.  **Run Unit Tests**:
    ```bash
    make test-unit
    ```

4.  **Run Basic E2E Tests**:
    Runs only the core scraping and writing logic. Useful for fast iteration and CI.
    ```bash
    make test-e2e-basic
    ```

5.  **Run a Demo Scrape**:
    Starts the stack and triggers a scrape job.
    
    Default (Hacker News, depth 1):
    ```bash
    make run
    ```

    Custom URL and Depth:
    ```bash
    make run URL=https://example.com DEPTH=2
    ```

## Development

-   **API**: Located in `api/`. Run locally with `uvicorn api.main:app --reload`.
-   **Scraper**: Located in `workers/scraper/`.
-   **Writer**: Located in `workers/writer/`.
-   **Image Extractor**: Located in `workers/image_extractor/`.
-   **Shared Library**: Located in `shared/`. Contains common Python clients and configuration.

### Linting & Formatting

The project uses several tools to ensure code quality:
-   **Black**: For deterministic code formatting.
-   **isort**: For import sorting (compatible with Black).
-   **Ruff**: For fast linting.
-   **Flake8**: For legacy style checks.
-   **Mypy**: For strict static type checking.
-   **Pylint**: For deep code analysis (Rating ≥ 9.5 required).

Run all checks:
```bash
make lint
```

Auto-format code:
```bash
make format
```

## Testing

The project emphasizes high test coverage:
-   **Unit Tests**: ~100% coverage for all components (API, Scraper, Writer, Image Extractor, Page Summarizer).
-   **E2E Tests**: Full integration tests using a local test runner and mock website.
    - **Reliable Verification**: Tests utilize a centralized polling mechanism that monitors the `GET /scrape` endpoint, waiting up to **5 minutes (300 seconds)** for a `COMPLETED` status to ensure all asynchronous background tasks (AI extraction, DB writes) have finished.
-   **Shared Library Tests**: Located in `tests/unit/shared/` for common client testing.

### AI Worker Testing

By default, the E2E tests use `LLM_PROVIDER=mock` to avoid external API calls and costs. This returns fixed "Mocked summary" and "Mocked explanation" results.

To test with real providers:
### Running with Real AI Services (e.g., OpenAI)

1.  Update `LLM_PROVIDER` in `docker-compose.yml` (e.g., to `openai`).
2.  Ensure `LLM_API_KEY` is set in your environment (it is passed to the workers via `docker-compose.yml`).
    ```bash
    export LLM_API_KEY=sk-...
    make run
    ```

## Retrieving Results

Since the system is event-driven, results are retrieved by polling the API or monitoring the job status.

### 1. Check Job Status & Get Data
Use the `scraping_id` returned by the `POST /scrape` endpoint.

```bash
curl http://localhost:8000/scrape?scraping_id=<ID> -H "X-API-Key: test-api-key-123"
```
Response (when `COMPLETED`):
```json
{
  "status": "COMPLETED",
  "scraping": { ... },
  "data": [
    {
      "url": "https://example.com/page1",
      "summary": "AI generated summary...",
      "images": [{"url": "...", "explanation": "..."}],
      "terms": [{"term": "foo", "frequency": 10}]
    }
  ]
}
```

### 2. Search Indexed Data
Search for websites containing a specific term:
```bash
curl http://localhost:8000/search?t=example -H "X-API-Key: test-api-key-123"
```

## License

MIT
