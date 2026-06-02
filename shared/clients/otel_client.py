"""
OpenTelemetry Client module providing a decorator for method/function observation,
and functions to initialize telemetry.
"""

import functools
import inspect
import logging
import os
from typing import Any, Callable, TypeVar

from opentelemetry import baggage, trace
from opentelemetry.baggage.propagation import W3CBaggagePropagator
from opentelemetry.propagate import set_global_textmap
from opentelemetry.propagators.composite import CompositePropagator
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import SpanProcessor, TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.trace.sampling import (
    ALWAYS_OFF,
    ALWAYS_ON,
    Decision,
    ParentBased,
    Sampler,
    SamplingResult,
    TraceIdRatioBased,
)
from opentelemetry.trace import Status, StatusCode
from opentelemetry.trace.propagation.tracecontext import (
    TraceContextTextMapPropagator,
)

# Setup logging
logger = logging.getLogger(__name__)

# Sensitive keywords as defined by guidelines/user request
SENSITIVE_KEYWORDS: set[str] = {
    "secret",
    "token",
    "key",
    "password",
    "credential",
    "auth",
    "sign",
}

# Type variables for decorating callables and classes
F = TypeVar("F", bound=Callable[..., Any])
C = TypeVar("C", bound=type)


class ErrorAwareSampler(Sampler):
    """
    A custom sampler that delegates to a ratio sampler,
    but returns RECORD_ONLY instead of DROP when the ratio sampler returns DROP.
    This allows us to inspect and export failed spans downstream.
    """

    def __init__(self, ratio: float):
        self._ratio_sampler = TraceIdRatioBased(ratio)

    def should_sample(
        self,
        parent_context: Any,
        trace_id: int,
        name: str,
        kind: Any = None,
        attributes: Any = None,
        links: Any = None,
        trace_state: Any = None,
    ) -> SamplingResult:
        result = self._ratio_sampler.should_sample(
            parent_context, trace_id, name, kind, attributes, links, trace_state
        )
        if result.decision != Decision.DROP:
            return result
        return SamplingResult(
            decision=Decision.RECORD_ONLY,
            attributes=result.attributes,
            trace_state=result.trace_state,
        )

    def get_description(self) -> str:
        return f"ErrorAwareSampler({self._ratio_sampler.get_description()})"


class ErrorAwareSpanProcessor(SpanProcessor):
    """
    A custom span processor that wraps another processor and only forwards spans
    that were sampled OR had a failure status (ERROR).
    """

    def __init__(self, delegate: SpanProcessor):
        self._delegate = delegate

    def on_start(self, span: Any, parent_context: Any = None) -> None:
        self._delegate.on_start(span, parent_context)

    def on_end(self, span: Any) -> None:
        is_sampled = span.context.trace_flags.sampled
        is_error = span.status.status_code == StatusCode.ERROR
        if is_sampled or is_error:
            self._delegate.on_end(span)

    def shutdown(self) -> None:
        self._delegate.shutdown()

    def force_flush(self, timeout_millis: int = 30000) -> bool:
        return bool(self._delegate.force_flush(timeout_millis))


def get_sampler_from_env() -> Sampler:
    """
    Parses OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG.
    Supports ErrorAwareSampler for ratio sampling to ensure failed spans
    are always captured.
    """
    sampler_type = os.getenv("OTEL_TRACES_SAMPLER", "always_on").lower()
    sampler_arg = os.getenv("OTEL_TRACES_SAMPLER_ARG", "1.0")

    try:
        ratio = float(sampler_arg)
    except ValueError:
        ratio = 1.0

    if sampler_type == "always_on":
        return ALWAYS_ON
    elif sampler_type == "always_off":
        return ALWAYS_OFF
    elif sampler_type == "traceidratio":
        return ErrorAwareSampler(ratio)
    elif sampler_type == "parentbased_always_on":
        return ParentBased(ALWAYS_ON)
    elif sampler_type == "parentbased_always_off":
        return ParentBased(ALWAYS_OFF)
    elif sampler_type == "parentbased_traceidratio":
        return ParentBased(ErrorAwareSampler(ratio))

    return ALWAYS_ON


class OTelLogFilter(logging.Filter):
    """
    Injects OTel trace_id, span_id, and correlation_id into log records.
    """

    def filter(self, record: logging.LogRecord) -> bool:
        span = trace.get_current_span()
        if span and span.get_span_context().is_valid:
            record.trace_id = format(span.get_span_context().trace_id, "032x")
            record.span_id = format(span.get_span_context().span_id, "016x")
        else:
            record.trace_id = "0" * 32
            record.span_id = "0" * 16

        # Pull correlation_id from baggage if present
        correlation_id = baggage.get_baggage("correlation_id")
        record.correlation_id = str(correlation_id) if correlation_id else ""
        return True


def init_telemetry(service_name: str) -> None:
    """
    Initializes OpenTelemetry TracerProvider and registers it globally.
    Configures an OTLP exporter if an endpoint is provided via environment variables.
    """
    if isinstance(trace.get_tracer_provider(), TracerProvider):
        # Already initialized
        return

    # Set the global composite textmap propagator
    set_global_textmap(
        CompositePropagator([TraceContextTextMapPropagator(), W3CBaggagePropagator()])
    )

    # Use standard resource attributes and the environment-configured sampler
    resource = Resource.create({"service.name": service_name})
    sampler = get_sampler_from_env()
    provider = TracerProvider(resource=resource, sampler=sampler)

    # Check for OTLP exporter endpoint
    otlp_endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if otlp_endpoint:
        try:
            # Dynamically import OTLP exporters so they are optional
            # dependencies in some contexts
            try:
                from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
                    OTLPSpanExporter,
                )
            except ImportError:
                # pylint: disable=line-too-long
                from opentelemetry.exporter.otlp.proto.http.trace_exporter import (  # type: ignore[assignment]
                    OTLPSpanExporter,
                )

            exporter = OTLPSpanExporter(
                endpoint=otlp_endpoint, insecure=True
            )  # type: ignore[call-arg]
            batch_processor = BatchSpanProcessor(exporter)
            provider.add_span_processor(ErrorAwareSpanProcessor(batch_processor))
            logger.info(
                "OpenTelemetry OTLP Exporter initialized with "
                "ErrorAwareSpanProcessor for service %s at %s",
                service_name,
                otlp_endpoint,
            )
        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Failed to initialize OpenTelemetry OTLP Exporter: %s", e)

    trace.set_tracer_provider(provider)

    # Automatically instrument asyncpg if available
    try:
        from opentelemetry.instrumentation.asyncpg import (
            AsyncPGInstrumentor,  # type: ignore[import-not-found]
        )

        AsyncPGInstrumentor().instrument()
        logger.info("Automatically instrumented asyncpg database queries.")
    except Exception as e:  # pylint: disable=broad-exception-caught
        logger.debug("asyncpg instrumentation skipped or failed: %s", e)


def __is_sensitive(name_or_val: str) -> bool:
    """
    Private helper to check if a key name or string value contains sensitive keywords.
    """
    lower_str = name_or_val.lower()
    return any(kw in lower_str for kw in SENSITIVE_KEYWORDS)


def __clean_value(val: str) -> str:
    """
    Private helper to redact values if they contain sensitive information.
    """
    if __is_sensitive(val):
        return "[REDACTED]"
    return val


def __clean_attributes(params: dict[str, Any]) -> dict[str, int | float | bool | str]:
    """
    Private helper to filter and clean attributes:
    1. Must be built-in types (int, float, bool, str).
    2. Redacts sensitive keys/values or skips them.
    """
    cleaned: dict[str, int | float | bool | str] = {}
    for k, v in params.items():
        k_str = str(k)
        if __is_sensitive(k_str):
            # Redact the value if the key itself contains a sensitive keyword
            if isinstance(v, (int, float, bool, str)) and not isinstance(v, type(None)):
                cleaned[k_str] = "[REDACTED]"
            continue

        # Check if v is an allowed built-in type (int, float, bool, str)
        # Note: bool is a subclass of int in Python, so isinstance(v, bool)
        # is caught under int, but we also explicitly check. We make sure None
        # is excluded.
        if isinstance(v, (int, float, bool, str)) and not isinstance(v, type(None)):
            if isinstance(v, str):
                cleaned[k_str] = __clean_value(v)
            else:
                cleaned[k_str] = v
    return cleaned


def __extract_span_attributes(
    func: Callable[..., Any], args: tuple[Any, ...], kwargs: dict[str, Any]
) -> dict[str, int | float | bool | str]:
    """
    Private helper to map args and kwargs to parameter names
    based on function signature.
    """
    try:
        sig = inspect.signature(func)
        bound = sig.bind(*args, **kwargs)
        bound.apply_defaults()
        params = dict(bound.arguments)
    except Exception:  # pylint: disable=broad-exception-caught
        # Fallback if signature binding fails
        params = {}
        for i, arg in enumerate(args):
            params[f"arg_{i}"] = arg
        params.update(kwargs)

    # Exclude self and cls from attributes
    params.pop("self", None)
    params.pop("cls", None)

    return __clean_attributes(params)


def observe(obj: Any = None, *, tracer_name: str | None = None) -> Any:
    """
    Decorator to trace a function, coroutine, or all public methods of a class.
    Attributes must be built-in types (int, float, bool, str) and sensitive keys/values
    are redacted.
    """
    if obj is None:
        return lambda o: observe(o, tracer_name=tracer_name)

    # Case 1: Decorator applied to a class
    if isinstance(obj, type):
        for name, attr in list(obj.__dict__.items()):
            # Observe public callables and methods
            if not name.startswith("_"):
                if callable(attr):
                    setattr(obj, name, observe(attr, tracer_name=tracer_name))
                elif isinstance(attr, (staticmethod, classmethod)):
                    underlying = attr.__func__
                    wrapped = observe(underlying, tracer_name=tracer_name)
                    if isinstance(attr, staticmethod):
                        setattr(obj, name, staticmethod(wrapped))
                    else:
                        setattr(obj, name, classmethod(wrapped))
        return obj

    # Case 2: Decorator applied to a function or method
    func = obj
    t_name = tracer_name or func.__module__ or "default"
    tracer = trace.get_tracer(t_name)
    span_name = func.__qualname__

    if inspect.iscoroutinefunction(func):

        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> Any:
            with tracer.start_as_current_span(span_name) as span:
                attrs = __extract_span_attributes(func, args, kwargs)
                for k, v in attrs.items():
                    span.set_attribute(k, v)
                correlation_id = baggage.get_baggage("correlation_id")
                if isinstance(correlation_id, str):
                    span.set_attribute("correlation_id", correlation_id)
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    span.record_exception(e)
                    span.set_status(Status(StatusCode.ERROR, str(e)))
                    raise

        return async_wrapper

    @functools.wraps(func)
    def sync_wrapper(*args: Any, **kwargs: Any) -> Any:
        with tracer.start_as_current_span(span_name) as span:
            attrs = __extract_span_attributes(func, args, kwargs)
            for k, v in attrs.items():
                span.set_attribute(k, v)
            correlation_id = baggage.get_baggage("correlation_id")
            if isinstance(correlation_id, str):
                span.set_attribute("correlation_id", correlation_id)
            try:
                return func(*args, **kwargs)
            except Exception as e:
                span.record_exception(e)
                span.set_status(Status(StatusCode.ERROR, str(e)))
                raise

    return sync_wrapper
