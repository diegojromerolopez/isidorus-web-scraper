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
| `IMAGES_BUCKET` | S3 bucket for images | `isidorus-images` |
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
