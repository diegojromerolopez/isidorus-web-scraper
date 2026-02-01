# Nube2e

Nube2e (Cloud-to-End) is a web scraping application designed to demonstrate a robust, event-driven architecture using LocalStack to emulate AWS services.

**DISCLAIMER: this project was created by making use of the agentic AI models Gemini 3.0 Pro/Gemini 3.0 Flash.**

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
    
    SQS_S --> Scraper[Scraper-Go]
    Scraper -->|3. Cycle Detection| Redis[(Redis)]
    Scraper -->|4. Track Depth| Redis
    Scraper -->|5. Found Image| SQS_I[SQS-Image Queue]
    Scraper -->|6. Page Data| SQS_W[SQS-Writer Queue]
    
    SQS_I --> Extractor[Image Extractor-Python]
    Extractor -->|7. Upload Image| S3[(S3-LocalStack)]
    Extractor -->|8. Explain via LangChain| LLM((AI Models))
    Extractor -->|9. Explanation Result| SQS_W
    
    SQS_W --> Writer[Writer-Go]
    Writer -->|10. Store Results| DB
    Writer -->|11. Complete| Redis
    Redis -->|12. Finalize| Writer
```

The system is built with a microservices approach:

1.  **API (FastAPI)**:
    -   Entry point for users.
    -   Initiates scraping jobs by sending messages to SQS.
    -   Tracks job status and results using Postgres and Redis.
    -   Provides endpoints to query results.

2.  **Scraper Worker (Go)**:
    -   Consumes scrape requests from SQS.
    -   Fetches and parses HTML.
    -   Extracts terms, links, and images.
    -   **Cycle Prevention**: Uses Redis Sets (`SADD`) to track and skip already-processed URLs per scraping session.
    -   **Distributed Tracking**: Uses Redis for distributed reference counting to track job completion.
    -   Recursively enqueues links for further scraping.

3.  **Image Extractor Worker (Python)**:
    -   Consumes image URLs found by the scraper.
    -   **S3 Persistence**: Downloads and uploads images to an AWS S3 bucket.
    -   **AI Explainer (LangChain)**: Generates image descriptions using various LLM providers (OpenAI, Gemini, Anthropic, Ollama, HuggingFace).
    -   Sends the S3 path and explanation to the Writer.

4.  **Writer Worker (Go)**:
    -   Consumes structured data (pages, terms, links, images, job completion events) from SQS.
    -   Writes data to PostgreSQL in a normalized schema.
    -   Handles job completion status updates.

## Configuration & Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `AWS_ENDPOINT_URL` | LocalStack URL | `http://localstack:4566` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@host:5432/db` |
| `REDIS_HOST` | Redis host | `localhost` or `redis` |
| `IMAGES_BUCKET` | S3 bucket for images | `nube2e-images` |
| `LLM_PROVIDER` | AI provider for explanations | `mock`, `openai`, `gemini`, etc. |
| `MAX_DEPTH` | Maximum recursive depth | `2` (Default from API) |

## API Endpoints

-   **`POST /scrape`**: Start a new scraping job.
    -   Body: `{"url": "...", "term": "...", "depth": 2}`
-   **`GET /scrape?scraping_id=1`**: Check status and get results of a scraping job.
-   **`GET /search?term={term}`**: Search for pages containing a specific term.
-   **`GET /terms`**: List all unique terms found across all scrapings.

## Infrastructure

The entire stack runs locally via Docker Compose:
-   **LocalStack**: Emulates SQS and S3.
-   **PostgreSQL**: Relational database for scraping results and image metadata.
-   **Redis**: In-memory store for cycle detection and job tracking counters.

## Technologies

-   **Backend**: Python 3.10+ (FastAPI), Go 1.21+
-   **ORM**: Tortoise ORM (API)
-   **Database**: PostgreSQL 15
-   **Caching/Coordination**: Redis 7
-   **Infrastructure**: LocalStack (AWS SQS/S3 emulation), Docker Compose
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

## Testing

The project emphasizes high test coverage:
-   **Unit Tests**: >90% coverage for all components.
-   **E2E Tests**: Full integration tests using a local test runner and mock website.

## License

MIT
