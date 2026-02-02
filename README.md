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
    API -->|2. Start Job| SQS_S[SQS-Scraper Queue]
    
    Admin((Admin)) -->|Manage Keys| AuthAdmin[Auth Admin-Django]
    AuthAdmin -->|3. Sync Keys| DB
    API -->|4. Validate Key| Redis[(Redis)]
    API -->|5. Check DB| DB
    
    SQS_S --> Scraper[Scraper-Go]
    Scraper -->|6. Cycle Detection| Redis
    Scraper -->|7. Track Depth| Redis
    Scraper -->|8. Found Image| SQS_I[SQS-Image Queue]
    Scraper -->|9. Page Data| SQS_W[SQS-Writer Queue]
    
    SQS_I --> Extractor[Image Extractor-Python]
    Extractor -->|10. Upload Image| S3[(S3-LocalStack)]
    Extractor -->|11. Explain via LangChain| LLM((AI Models))
    Extractor -->|12. Explanation Result| SQS_W

    Scraper -->|9b. Page Text| SQS_PS[SQS-Summarizer Queue]
    SQS_PS --> Summarizer[Page Summarizer-Python]
    Summarizer -->|13. Summarize| LLM
    Summarizer -->|14. Summary Result| SQS_W
    
    SQS_W --> Writer[Writer-Go]
    Writer -->|15. Store Results| DB
    Writer -->|16. Complete| Redis
    Redis -->|17. Finalize| Writer
```

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

2.  **Run End-to-End Tests**:
    ```bash
    make test-e2e
    ```

3.  **Run Unit Tests**:
    ```bash
    make test-unit
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
-   **Ruff**: For fast linting and import sorting.
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
-   **Unit Tests**: 100% coverage for all components (API, Scraper, Writer, Image Extractor, Page Summarizer).
-   **E2E Tests**: Full integration tests using a local test runner and mock website.
-   **Shared Library Tests**: Located in `tests/unit/shared/` for common client testing.

## License

MIT
