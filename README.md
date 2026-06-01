<div align="center">
  <img src="docs/images/logo.png" alt="Isidorus Web Scraper Logo" width="300"/>
</div>

# Isidorus Web Scraper

[![Unit Tests](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/tests-unit.yml/badge.svg)](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/tests-unit.yml)
[![Python Lint](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/python-lint.yml/badge.svg)](https://github.com/diegojromerolopez/isidorus-web-scraper/actions/workflows/python-lint.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Python 3.14+](https://img.shields.io/badge/python-3.14+-blue.svg)](https://www.python.org/downloads/)
[![Go 1.24+](https://img.shields.io/badge/go-1.24+-00ADD8.svg)](https://golang.org/dl/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.100+-009688.svg)](https://fastapi.tiangolo.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-336791.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D.svg)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED.svg)](https://www.docker.com/)
[![LocalStack](https://img.shields.io/badge/LocalStack-AWS-4D4D4D.svg)](https://localstack.cloud/)
[![Code style: black](https://img.shields.io/badge/code%20style-black-000000.svg)](https://github.com/psf/black)
[![Ruff](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/astral-sh/ruff/main/assets/badge/v2.json)](https://github.com/astral-sh/ruff)
[![Imports: isort](https://img.shields.io/badge/%20imports-isort-%231674b1?style=flat&labelColor=ef8336)](https://pycqa.github.io/isort/)

**Isidorus Web Scraper** is named after **Isidore of Seville** (c. 560-636 AD), the renowned scholar and Archbishop of Seville who compiled the *Etymologiae*, the first encyclopedia of all human knowledge. Known as "The Schoolmaster of the Middle Ages," Isidore meticulously gathered, organized, and preserved the wisdom of his time. Just as Isidore collected and systematized knowledge, Isidorus Web Scraper archives and indexes web content with precision and thoroughness.

This project is an **AI-driven web scraping and content analysis platform**. It serves as an **excellent showcase of LocalStack**, demonstrating how to build, test, and orchestrate a complex AWS-based architecture entirely on a local machine without incurring cloud costs.

**DISCLAIMER: this project was created by making use of the agentic AI models Gemini 3.0 Pro/Gemini 3.0 Flash/Claude Sonnet 4.5.**

## Overview

The application takes a URL, recursively scrapes the website to a configurable depth, and stores:
- **Pages**: Metadata of visited URLs and **AI-generated summaries**.
- **Searchable Content**: Full-text index of content and summaries in **OpenSearch**.
- **Links**: Graph of internal and external links.
- **Images**: Extracted image URLs, stored in **S3**, with **AI-generated descriptions** (Alt-text).

---

## 🤖 AI-First Analysis

Isidorus goes beyond simple scraping by integrating AI into the heart of its processing pipeline:

- **Intelligent Summarization**: Automatically distills long-form web content into concise, readable summaries.
- **Computer Vision (Alt-Text Generation)**: Processes images found during scraping to generate descriptive text, making visual content searchable and accessible.
- **Local LLM Support**: Defaults to **Ollama** (`tinyllama`) for privacy-conscious, local-first inference, but supports OpenAI and other providers via LangChain.

## ☁️ LocalStack Showcase (E2E & Dev)

This project demonstrates how to use **LocalStack** to emulate a complete AWS environment locally. This is the heart of our **End-to-End (E2E) testing** strategy, allowing for high-fidelity verification without cloud costs.

```mermaid
graph TD
    subgraph "Testing Infrastructure (LocalStack)"
        LS[LocalStack]
        S3_L[S3 Bucket]
        SQS_L[SQS Queues]
        DDB_L[DynamoDB Table]
        LS --- S3_L
        LS --- SQS_L
        LS --- DDB_L
    end

    TR[Test Runner] -->|Start Scrape| API[API]
    API -->|Enqueue| SQS_L
    API -->|Status| DDB_L
    
    Scraper[Scraper Worker] -->|Consume| SQS_L
    Scraper -->|Extract| Site[Mock Website]
    
    Extractor[Image Extractor] -->|Upload| S3_L
    Summarizer[Page Summarizer] -->|Mock AI| OllamaMock[Ollama Mock]
```

- **Zero-Cloud Architecture**: Emulates SQS, S3, and DynamoDB, allowing for a 1:1 local-to-cloud development experience.
- **Rapid Iteration**: Test complex event-driven workflows (like asynchronous image processing) instantly without waiting for cloud provisioning.
- **E2E Testing Fidelity**: Uses real AWS SDKs (`aioboto3`, `boto3`, AWS Go SDK) against high-fidelity mocks, ensuring production-ready code.

## 🏗️ Provider-Agnostic Infrastructure (Production)

Isidorus is strictly **provider-agnostic**. While it can run on AWS, it can also be deployed entirely on self-hosted, open-source infrastructure replacing those cloud services. This is showcased in our **Production Stack** (`make prod-up`).

```mermaid
graph TD
    subgraph "Agnostic Infrastructure (Self-Hosted)"
        Minio[Minio - S3 Compatible]
        EMQ[ElasticMQ - SQS Compatible]
        Scylla[ScyllaDB - DynamoDB Compatible]
    end

    User((User)) -->|HTTP| API[API]
    API -->|S3_ENDPOINT_URL| Minio
    API -->|SQS_ENDPOINT_URL| EMQ
    API -->|DYNAMODB_ENDPOINT_URL| Scylla

    Workers[Workers Pool] --> Minio
    Workers --> EMQ
    Workers --> Scylla
    
    AI[Self-Hosted AI] -->|Local Inference| Ollama[Ollama - tinyllama]
    Workers --> Ollama
```

- **Endpoint Agnosticism**: By using `BASE_ENDPOINT_URL` (or service-specific overrides), the application can talk to any S3/SQS/DynamoDB compatible API.
- **Cloud-Native & Hybrid**: Deploy on Kubernetes using local storage/queues or mix-and-match with managed cloud services.
- **Self-Hosted AI**: Integration with **Ollama** ensures even the LLM processing is completely decoupled from external vendors.

## Architecture

```mermaid
graph TD
    User((User)) -->|HTTP| Frontend[Frontend-React]
    Frontend -->|HTTP API Requests| API[API-FastAPI]
    
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
    
    SQS_I --> Extractor[Image Extractor-Go]
    Extractor -->|11. Upload Image| S3[(S3-LocalStack)]
    Extractor -->|12. Meta Result| SQS_W
    Extractor -->|13. Explain Request| SQS_IE[SQS-Explainer Queue]
    
    SQS_IE --> Explainer[Image Explainer-Python]
    Explainer -->|14. Download Image| S3
    Explainer -->|15. Explain via LangChain| LLM((AI Models))
    Explainer -->|16. Explanation Result| SQS_W

    Scraper -->|10b. Page Text| SQS_PS[SQS-Summarizer Queue]
    SQS_PS --> Summarizer[Page Summarizer-Python]
    Summarizer -->|17. Summarize| LLM
    Summarizer -->|18. Summary Result| SQS_W
    Summarizer -->|19. Index Request| SQS_Index[SQS-Indexer Queue]
    
    SQS_Index --> Indexer[Indexer-Go]
    Indexer -->|20. Index Content| OS[(OpenSearch)]

    SQS_W --> Writer[Writer-Go]
    Writer -->|21. Store Results| DB
    Writer -->|22. Mark Completion| Dynamo

    API -->|23. Global Search| OS
    API -->|24. Enqueue Deletion| SQS_D[SQS-Deletion Queue]
    SQS_D --> Deletion[Deletion Worker-Python]
    Deletion -->|25. Delete Objects| S3
    Deletion -->|26. Delete Data| DB
    Deletion -->|27. Delete Meta| Dynamo
```

## Data Storage Strategy

Isidorus uses a hybrid storage approach to optimize for both relational integrity and high-throughput status monitoring:

### 1. PostgreSQL (Relational Site Content)
**Location**: `scrapings`, `scraped_pages`, `page_links`, `page_images`.
**Purpose**: Stores the core "knowledge graph" extracted from the web. 
**Why**: 
- **Relational Integrity**: Perfect for the complex relationships between pages, images, and links.
- **Identity**: Acts as the system's "Identity Store" by generating unique incremental IDs for jobs.

### 2. OpenSearch (Full-Text Search)
**Location**: `scraped_pages` index.
**Purpose**: Enables high-performance, relevance-based global search across all scraped content and summaries.
**Why**:
- **Relevance Scoring**: Provides better search results than simple SQL LIKE or term counting.
- **Scalability**: Handles large volumes of text data efficiently.
- **Highlights**: Supports returning snippets of matching content.

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

0.  **Frontend (React 18)**:
    -   Modern SPA served by Node.js.
    -   Provides a Dashboard for initiating and monitoring scraping jobs.
    -   Features detailed views for scraping results, including term stats, AI summaries, and **collapsible page reports** for easy scannability.
    -   **Improved Navigation**: Clickable URLs in the history table allow direct access to job details.
    -   Interacts with the API via strictly typed TS methods.

1.  **Auth Admin (Django)**:
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

3.  **Image Extractor Worker (Go)**:
    -   Consumes image URLs from `image-extractor-queue`.
    -   **S3 Persistence**: Downloads and uploads images to an AWS S3 bucket.
    -   Sends image metadata to the Writer and enqueues tasks for the Image Explainer.

4.  **Image Explainer Worker (Python)**:
    -   Consumes tasks from `image-explainer-queue`.
    -   **S3 Retrieval**: Downloads images from AWS S3 for processing.
    -   **AI Explainer**: Generates image descriptions (alt-text) using LLM providers.
    -   Sends explanation results to the Writer.

5.  **Page Summarizer Worker (Python)**:
    -   Consumes text content from `page-summarizer-queue`.
    -   **AI Summarization**: Generates concise summaries of web pages using LLMs.
    -   Sends results to the Writer and enqueues indexing requests.

6.  **Indexer Worker (Go)**:
    -   Consumes data from `indexer-queue`.
    -   **OpenSearch Indexing**: Indexes page content and summaries for global search.

7.  **Writer Worker (Go)**:
    -   Consumes structured data (pages, terms, links, images, job completion events) from SQS.
    -   Writes data to PostgreSQL in a normalized schema.
    -   Handles job completion status updates.

8.  **Deletion Worker (Python)**:
    -   Consumes deletion requests from `deletion-queue`.
    -   **Batched Deletion**: Efficiently removes large datasets from PostgreSQL, S3, and OpenSearch.
    -   Cleanly removes job metadata from DynamoDB.

9.  **Shared Library (Python)**:
    -   Common package (`shared/`) containing reusable components.
    -   **Async AWS Clients**: `SQSClient` and `S3Client` using `aioboto3` for non-blocking I/O.
    -   **Configuration**: Base `Configuration` class for centralized environment variable management.
    -   Used by both API and Image Extractor worker to ensure consistency.

## Configuration & Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `BASE_ENDPOINT_URL` | Base service endpoint fallback | `http://localstack:4566` |
| `S3_ENDPOINT_URL` | Specific S3 endpoint override | `http://minio:9000` |
| `SQS_ENDPOINT_URL` | Specific SQS endpoint override | `http://elasticmq:9324` |
| `DYNAMODB_ENDPOINT_URL`| Specific DynamoDB endpoint override | `http://scylla:8042` |
| `DATABASE_URL` | Postgres Connection String | `postgres://user:pass@host:5432/db` |
| `REDIS_HOST` | Redis host | `localhost` or `redis` |
| `IMAGE_BUCKET` | S3 bucket for images | `isidorus-images` |
| `LLM_PROVIDER` | AI provider for explanations | `mock`, `openai`, `gemini`, etc. |
| `SCRAPER_REPLICAS` | Number of Scraper instances | `3` |
| `WRITER_REPLICAS` | Number of Writer instances | `2` |
| `IMAGE_EXTRACTOR_REPLICAS`| Number of Extractor instances | `3` |
| `IMAGE_EXPLAINER_REPLICAS`| Number of Explainer instances | `4` |
| `PAGE_SUMMARIZER_REPLICAS`| Number of Summarizer instances | `2` |
| `INDEXER_REPLICAS` | Number of Indexer instances | `1` |
| `DELETION_REPLICAS` | Number of Deletion instances | `1` |

## Horizontal Scaling

The application is designed for horizontal scalability. Depending on your environment, scaling is handled differently:

### 🐳 Docker Compose (Manual)
You can manually adjust the "funnel" of your scraping pipeline by setting the number of replicas in your environment or using the `--scale` flag.

| Worker | Default Replicas | Role |
|--------|------------------|------|
| **Scraper** | 3 | High-throughput Go crawler. |
| **Extractor** | 3 | Network-intensive S3 heavy lifting. |
| **Writer** | 2 | Concurrent DB persistence. |
| **Indexer** | 1 | OpenSearch indexing. |
| **Explainer** | 4 | AI processing (The "Bottleneck"). |
| **Summarizer** | 2 | AI summarization. |

Example for scaling up the AI explainer:
```bash
IMAGE_EXPLAINER_REPLICAS=10 docker compose up -d --scale image-explainer-worker=10
```

### ☸️ Kubernetes (Automatic via KEDA)
In Kubernetes, scaling is **event-driven and automatic**. The cluster uses [KEDA](https://keda.sh/) to monitor SQS queue depths and scale workers proportionally to the workload (even to zero when idle). 

Refer to the [Event-Driven Autoscaling (KEDA)](#-event-driven-autoscaling-keda) section for details on replica boundaries.

## API Endpoints

-   **`POST /scrape`**: Start a new scraping job.
    -   Body: `{"url": "...", "depth": 2}`
    -   **Example**:
        ```bash
        curl -X POST http://localhost:8000/scrape \
          -H "Content-Type: application/json" \
          -H "X-API-Key: test-api-key-123" \
          -d '{"url": "https://example.com", "depth": 1}'
        ```
        Response:
        ```json
        {"scraping_id": 123}
        ```
-   **`GET /scraping/{id}`**: Check status and get results of a scraping job.
-   **`DELETE /scraping/{id}`**: Delete a scraping job and all its related data.
-   **`GET /search?t={term}`**: Global full-text search across all content and summaries using OpenSearch.

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

### 🐳 Docker Compose Architecture

The project uses a modular Docker Compose setup to support multiple environments without duplication:

-   **`docker-compose.base.yml`**: Defines common services (API, Workers, DBs) and builds.
-   **`docker-compose.yml`**: **Development** overrides. Adds LocalStack, Ollama, and host ports.
-   **`docker-compose.prod.yml`**: **Production** overrides. Adds Minio, ScyllaDB, ElasticMQ.
-   **`docker-compose.e2e.yml`**: **Testing** overrides. Adds Test Runner and Mocks.

**Note**: The `Makefile` handles the complex file chaining for you (e.g., `docker compose -f docker-compose.base.yml -f ...`).

### Production-Ready Infrastructure (Optional)
You can switch to a more production-aligned stack using `make prod-up`. This replaces LocalStack with:
-   **Minio**: S3-compatible object storage.
-   **ElasticMQ**: Standalone SQS-compatible queue system.
-   **ScyllaDB (Alternator)**: High-performance DynamoDB-compatible NoSQL store.

## Technologies

-   **Frontend**: React 18, TypeScript, TailwindCSS, Vite
-   **Backend**: Python 3.14+ (FastAPI), Go 1.24+
-   **Async I/O**: `redis.asyncio` (async Redis client), `aioboto3` (async AWS SDK), `httpx` (async HTTP client)
-   **ORM**: Tortoise ORM (API)
-   **Database**: PostgreSQL 15
-   **Caching/Coordination**: Redis 7
-   **Infrastructure**: LocalStack (AWS SQS/S3/DynamoDB emulation), Docker Compose
-   **NoSQL**: DynamoDB (Job History)
-   **AI**: LangChain (Multi-provider support)
-   **Testing**: `unittest` (Python), Vitest (Frontend), `go test` (Go), `boto3`/`requests` (E2E)
-   **Mock Website**: A static site container for safe, deterministic E2E testing.

## Prerequisites

-   Docker & Docker Compose (v2+)
-   Python 3.14+
-   Go 1.24+
-   Make
-   [Kind](https://kind.sigs.k8s.io/) (for local Kubernetes)
-   [kubectl](https://kubernetes.io/docs/tasks/tools/)

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

5.  **Run on Kubernetes (Kind)**:
    Deploy the entire stack to a local [Kind](https://kind.sigs.k8s.io/) cluster.
    ```bash
    # 1. Automated setup of cluster, images, and manifests
    make k8s-setup-all

    # 2. Access the application (Port-forwarding)
    make k8s-port-forward
    ```
    Access the Frontend at **http://localhost:3000** and the API at **http://localhost:8000**.

6.  **Cloud Deployment (Production)**:
    For production deployments (EKS, GKE, AKS), refer to our [Cloud Kubernetes Deployment Plan](file:///Users/diegoj/.gemini/antigravity/brain/934f552a-c11e-4b2d-9ca2-aa92695a8df0/cloud_implementation_plan.md).
    
    You can use the helper script to prepare your local manifests for a remote registry:
    ```bash
    bash scripts/k8s-cloud-prepare.sh <your-registry-url>
    ```

### ☸️ Kubernetes Operations

The following commands are available for managing the local Kind cluster:

| Command | Description |
|---------|-------------|
| `make k8s-setup-all` | Full automated setup: cluster creation, image build/load, and deployment. |
| `make k8s-port-forward` | Forwards Frontend (3000) and API (8000) to your local machine. |
| `make k8s-deploy` | Re-applies all manifests to the cluster (useful for rapid manifest testing). |
| `make k8s-secrets` | Generates and applies Kubernetes secrets using local environment variables. |
| `make k8s-update-images` | Rebuilds and re-loads images into the cluster without recreating it. |

### ⚠️ Kubernetes Production Readiness

The current Kubernetes manifests are designed for a **High-Fidelity Local Environment** (Kind) and are **NOT** fully production-ready. 

**Current Limitations & Required Changes for Production:**

1.  **Single Point of Failure**: The current setup runs on `kind` (Single Node). A production cluster must run on **3+ physical nodes** across multiple Availability Zones (AZs).
2.  **Affinity Rules**: The current manifests do NOT enforce `podAntiAffinity`. In production, you must add these rules to ensure replicas are scheduled on *different* physical nodes.
3.  **Manual Orchestration**: We use custom scripts (`k8s/infra/scripts/pg-replication.sh`) for basic Master/Slave replication. For production, use **Operators** (e.g., [CloudNativePG](https://cloudnative-pg.io/), [Scylla Operator](https://operator.scylladb.com/)) to handle automated failover, backups, and recovery.
    > **Note**: Infrastructure manifests are now managed via **Kustomize** (`kubectl apply -k k8s/infra/`) to dynamically load these scripts from `k8s/infra/scripts/`.
4.  **Resource Limits**: CPU/Memory requests and limits are not strictly enforced to allow for flexible local development. In production, these **must** be defined to prevent "noisy neighbor" issues.

### 🚀 Event-Driven Autoscaling (KEDA)

The worker pool is configured for **Event-Driven Autoscaling** using [KEDA](https://keda.sh/). Instead of scaling based on CPU, workers scale based on SQS queue depth:

- **Scraper**: Scales between **1 and 10** replicas (target: 5 messages per pod).
- **Image Extractor**: Scales between **0 and 10** replicas (target: 5 messages per pod).
- **Writer**: Scales between **1 and 5** replicas (target: 10 messages per pod).
- **Indexer**: Scales between **1 and 5** replicas (target: 10 messages per pod).
- **Ollama (Inference)**: Scales between **1 and 4** replicas based on the combined load of the AI queues.
- **Image Explainer**: Scales between **0 and 8** replicas (target: 1 message per pod).
- **Page Summarizer**: Scales between **0 and 5** replicas (target: 1 message per pod).

> [!NOTE]
> AI workers scale to **0** when idle to save local resources, while the scraper always keeps **1** pod ready for immediate responsiveness.

**Monitor Autoscaling**:
```bash
kubectl get horizontalpodautoscaler --watch -n isidorus
```

**Clean Up**: To delete the Kind cluster and stop all Kubernetes resources:
```bash
kind delete cluster --name isidorus
```

7.  **Run a Demo Scrape**:
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

-   **Frontend**: Located in `frontend/`. Run locally with `cd frontend && npm run dev` (Access at `http://localhost:3000`).
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
-   **Unit Tests**: ~100% coverage for all components (API, Frontend, Scraper, Writer, Image Extractor, Page Summarizer).
    - **Frontend**: Tested using **Vitest** and **React Testing Library**.
-   **E2E Tests**: Full integration tests using a local test runner and mock website.
    - **Reliable Verification**: Tests utilize a centralized polling mechanism that monitors the `GET /scraping/{id}` endpoint, waiting up to **5 minutes (300 seconds)** for a `COMPLETED` status to ensure all asynchronous background tasks (AI extraction, DB writes) have finished.
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
curl http://localhost:8000/scraping/<ID> -H "X-API-Key: test-api-key-123"
```
Response (when `COMPLETED`):
```json
{
  "scraping": {
    "id": 123,
    "url": "https://example.com",
    "status": "COMPLETED",
    "created_at": "...",
    "completed_at": "...",
    "depth": 1,
    "pages": [
      {
        "url": "https://example.com/page1",
        "summary": "AI generated summary...",
        "images": [{"url": "...", "explanation": "..."}]
      }
    ]
  }
}
```

### 2. Search Indexed Data
Search for websites containing a specific term:
```bash
curl http://localhost:8000/search?t=example -H "X-API-Key: test-api-key-123"
```

## 🚀 Pending Developments (TODO)

This project is a functional showcase, but there are several areas planned for "Production-Grade" evolution:

- **📊 Observability**:
    - [x] Integration with **OpenTelemetry** (Fully Implemented).
    - **Structured Logging** using `slog` for better observability and correlation.
    - Centralized logging with **Prometheus/Grafana** dashboards for worker health and queue depths.
- **🛡️ Resilience**:
    - Implementation of **Dead Letter Queues (DLQ)** for handling failed scrapes or AI processing errors.
    - Advanced **Backpressure** mechanisms in Go workers to handle traffic spikes.
    - Retry strategies with exponential backoff for external LLM calls.
- **🔒 Security**:
    - **Identity Provider (IdP)** integration for the Auth Admin.
    - Multi-tenant data isolation at the database and OpenSearch layers.
    - Encryption at rest for S3 objects and database fields.
- **🏗️ Production Infrastructure**:
    - Native `docker-compose.prod.yml` replacing LocalStack with dedicated services like **Minio** (S3), **RabbitMQ/NATS** (SQS alternative), or native AWS/GCP/Azure services.
    - **Cloud Kubernetes**: Leverages the existing K8s manifests and KEDA autoscaling for enterprise-grade scalability.
- **⚡ Performance**:
    - Moving more workers to **Go** where sub-millisecond I/O is critical.
    - Vector database integration for semantic search beyond keyword matching.

## 📊 Observability & Distributed Tracing (OpenTelemetry)

Isidorus includes high-fidelity distributed tracing designed around **Domain-Driven Design (DDD)** guidelines. All raw OpenTelemetry library interactions are fully decoupled within concrete infrastructure clients, keeping core business/domain layers completely stable and unpolluted.

### 🌟 Key Tracing Features
1. **Full Span Coverage**: Every function and method across both Go and Python services, repositories, and clients is wrapped in an active OpenTelemetry span.
2. **Automatic Parameter Mapping**: Function parameters are dynamically extracted and set as span attributes if they are built-in types (`int`, `float`, `bool`, `string`).
3. **Sensitive Keyword Redaction**: Any parameter name or string value containing authentication or private information is automatically redacted to prevent secret leakage in trace collectors. Flagged terms include:
   `secret`, `token`, `key`, `password`, `pass`, `auth`, `credential`, `private`, `cert`, `jwt`, `conn`, `access`, `sign`
4. **Local OTel Collector Pipeline**: Traces are exported over OTLP (gRPC on port `4317` and HTTP on port `4318`) to a local OpenTelemetry Collector service which prints detailed traces to standard console output.

### ⚙️ Infrastructure Integrations
- **Docker Compose**: The `otel-collector` service is configured in `docker-compose.base.yml`. All services across `docker-compose.yml` and `docker-compose.prod.yml` automatically inherit `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317`.
- **Kubernetes**: Deployed via custom manifests in `k8s/infra/` (`otel-collector-configmap.yaml`, `otel-collector-deployment.yaml`, `otel-collector-service.yaml`) and registered in the `kustomization.yaml`. FQDN endpoint `http://otel-collector.isidorus.svc.cluster.local:4317` is cleanly injected across all application deployments under `k8s/apps/`.

### 🧪 Unit Testing Span Isolation
Unit tests in both languages verify trace capture in isolation:
- **Python**: Tests in `tests/unit/shared/clients/test_otel_client.py` use an `InMemorySpanExporter` and clean up the context after each test via `OtelClient.clear_spans()`.
- **Go**: Repository and service tests utilize OpenTelemetry's `go.opentelemetry.io/otel/sdk/trace/tracetest` package to verify span properties, resetting the collector between tests via `exporter.Reset()`.

## License

MIT
