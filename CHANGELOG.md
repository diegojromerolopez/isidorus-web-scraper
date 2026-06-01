# Changelog

All notable changes to the **Isidorus Web Scraper** project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [0.2.0] - 2026-06-01

### Added
- **Full End-to-End Tracing Instrumentation**: Integrated standard OpenTelemetry (OTel) tracing across the entire codebase (FastAPI backend and all Golang/Python workers).
- **Python OTel Client Implementation**: Introduced `shared/clients/otel_client.py` to handle standard tracing configurations, span injection, and automatic context parsing.
- **Go OTel Client Implementation**: Implemented robust Go OTel wrapper libraries (`otel_client.go`) within each Go worker (`scraper`, `writer`, `image_extractor`, `indexer`) to standardize trace execution.
- **Distributed Context Propagation via SQS**: Enabled automatic propagation of trace context (`traceparent` and `baggage`) across SQS messaging queues using SQS message attributes.
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
