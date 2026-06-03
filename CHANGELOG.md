# Changelog

All notable changes to the **Isidorus Web Scraper** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.2.0] - 2026-06-01

### Added
- **Deep Database Query Tracing**: Integrated GORM OpenTelemetry plugin (`"gorm.io/plugin/opentelemetry/tracing"`) in Go's `writer` worker and `opentelemetry.instrumentation.asyncpg` in Python's `api` backend to trace database transactions and PostgreSQL queries automatically.
- **Outgoing HTTP Tracing (`otelhttp`)**: Wrapped Go outbound HTTP clients with `"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"` inside the `scraper` and `image_extractor` workers to trace network latency, handshake times, and DNS resolution.
- **OpenSearch Client Tracing**: Wrapped the OpenSearch client's transport layer with `otelhttp.NewTransport` inside the `indexer` worker to automatically record indexing operations as child spans.
- **Prometheus OTLP Metrics push Integration**: Enabled standard OpenTelemetry Meter Providers pushing metrics to OTel Collector, exposed port `8889` as a Prometheus scrapable endpoint inside the collector config, spun up a local `prometheus` instance in `docker-compose.yml`, and registered standard meter counters (`scraping_jobs_processed_total`) in the Go scraper loop.
- **Error-Aware Tail Sampling Processor (Python & Go)**: Implemented next-level tail sampling that captures 25% of successful trace spans probabilistically (based on `OTEL_TRACES_SAMPLER` and `OTEL_TRACES_SAMPLER_ARG`) while guaranteeing that 100% of failed (error) spans are always retained and exported.
- **Trace-Log Context Correlation (MDC Log Filters)**: Integrated standard Python log filters (`OTelLogFilter`) and Go logging wrappers to automatically extract and inject `trace_id`, `span_id`, and `correlation_id` from the active span context into stdout/stderr logs.
- **Shared OTLP Trace Exporter Library**: Standardized OTLP gRPC trace exporting in a unified `shared/go/telemetry` library, updating all Go workers (`scraper`, `writer`, `indexer`, `image_extractor`) to dynamically configure OTLP exporters when `OTEL_EXPORTER_OTLP_ENDPOINT` is present.
- **Standard SQS Message Attributes Context Propagation**: Refactored SQS messaging components in both Go and Python to transition from custom JSON body propagation (`_trace_context`) to standard SQS `MessageAttributes`, ensuring robust enterprise tracing patterns.
- **Full End-to-End Tracing Instrumentation**: Integrated standard OpenTelemetry (OTel) tracing across the entire codebase (FastAPI backend and all Golang/Python workers).
- **Python OTel Client Implementation**: Introduced `shared/clients/otel_client.py` to handle standard tracing configurations, span injection, and automatic context parsing.
- **Go OTel Client Implementation**: Implemented robust Go OTel wrapper libraries (`otel_client.go`) within each Go worker (`scraper`, `writer`, `image_extractor`, `indexer`) to standardize trace execution.
- **Baggage Correlation ID Tracking**: Added support for custom baggage context (e.g., `correlation_id`) propagating seamlessly from the client request down to deep worker operations.
- **Standard OpenTelemetry Collector**: Added standard OTel Collector deployment config mapping standard ports (`4317` gRPC, `4318` HTTP, `13133` Health Check) to output spans to `stdout` (`debug` and `logging` verbosity).
- **Kubernetes Observability Manifests**: Created declarative Kubernetes templates under `k8s/infra/` for launching the OTel Collector Deployment, Service, and ConfigMap in clustered environments.
- **Comprehensive Unit Testing**: Achieved extremely high unit test coverage with dedicated mock suites for the Go/Python OTel clients and SQS context-propagation routines.

### Changed
- Refactored all data stores, API endpoints, worker tasks, and messaging clients to standardise trace span creation (`SpanStart`/`SpanEnd` or context decorators).
- Updated local Docker Compose stacks ([docker-compose.base.yml](file:///Users/diegoj/repos/isidorus-web-scraper/docker-compose.base.yml), [docker-compose.yml](file:///Users/diegoj/repos/isidorus-web-scraper/docker-compose.yml), [docker-compose.prod.yml](file:///Users/diegoj/repos/isidorus-web-scraper/docker-compose.prod.yml)) to run standard `otel-collector` instead of SigNoz for lightweight logging.
- Set SemVer version dynamically in `.semver` to `0.2`.
- **Python Type Safety & Mypy Resolution**: Corrected 20 strict static type checking and annotation warnings across 6 Python files, including missing return types, Django and Tortoise ORM dynamic property definitions, and S3 batch deletion flat values mapping using standard `typing.cast`.
- **Pylint Score Enforcement**: Integrated automatic `--fail-under=9.5` threshold checks into the `pylint` target in the `Makefile` to align static verification with project quality standards.

### Removed
- **Flake8 Linter**: Fully removed the legacy `flake8` dependency, its configurations (`.flake8`), local make targets, documentation references, and CI workflows (`python-lint.yml`) in favor of `ruff` as the single, fast pep8 engine.

---

## [0.1.0] - 2026-05-31

### Added
- Initial project architecture with FastAPI API and Golang/Python workers.
- Core relational graph database storage schema on PostgreSQL.
- Distributed index engine integration using OpenSearch.
- LocalStack integration for mocking SQS, S3, and DynamoDB.
- Basic Kind + KEDA autoscaling Kubernetes setup.
