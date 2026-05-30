"""
OpenTelemetry Client module providing a decorator for method/function observation,
and functions to initialize telemetry.
"""

import functools
import inspect
import logging
import os
from typing import Any, Callable, TypeVar

from opentelemetry import trace
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace import Status, StatusCode

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


def init_telemetry(service_name: str) -> None:
    """
    Initializes OpenTelemetry TracerProvider and registers it globally.
    Configures an OTLP exporter if an endpoint is provided via environment variables.
    """
    if isinstance(trace.get_tracer_provider(), TracerProvider):
        # Already initialized
        return

    # Use standard resource attributes
    resource = Resource.create({"service.name": service_name})
    provider = TracerProvider(resource=resource)

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
                from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter  # type: ignore

            exporter = OTLPSpanExporter(
                endpoint=otlp_endpoint, insecure=True
            )  # type: ignore[call-arg]
            provider.add_span_processor(BatchSpanProcessor(exporter))
            logger.info(
                "OpenTelemetry OTLP Exporter initialized for service %s at %s",
                service_name,
                otlp_endpoint,
            )
        except Exception as e:  # pylint: disable=broad-exception-caught
            logger.error("Failed to initialize OpenTelemetry OTLP Exporter: %s", e)

    trace.set_tracer_provider(provider)


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
                try:
                    return await func(*args, **kwargs)
                except Exception as e:
                    span.record_exception(e)
                    span.set_status(Status(StatusCode.ERROR, str(e)))
                    raise

        return async_wrapper
    else:

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> Any:
            with tracer.start_as_current_span(span_name) as span:
                attrs = __extract_span_attributes(func, args, kwargs)
                for k, v in attrs.items():
                    span.set_attribute(k, v)
                try:
                    return func(*args, **kwargs)
                except Exception as e:
                    span.record_exception(e)
                    span.set_status(Status(StatusCode.ERROR, str(e)))
                    raise

        return sync_wrapper
