"""
Unit tests for OpenTelemetry client and decorator.
"""

import asyncio
import unittest

from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)

from shared.clients.otel_client import observe


class TestOTelClient(unittest.TestCase):
    """
    Tests the OpenTelemetry @observe decorator.
    """

    def setUp(self) -> None:
        from unittest.mock import patch

        # Setup clean TracerProvider for testing
        self.provider = TracerProvider()
        self.exporter = InMemorySpanExporter()
        self.provider.add_span_processor(SimpleSpanProcessor(self.exporter))

        # Patch trace.get_tracer to always return a tracer from our test provider
        self.patcher = patch(
            "opentelemetry.trace.get_tracer", side_effect=self.provider.get_tracer
        )
        self.mock_get_tracer = self.patcher.start()

    def tearDown(self) -> None:
        self.patcher.stop()

    def test_observe_sync_function(self) -> None:
        """
        Verify sync function span creation and parameter filter/redaction.
        """

        @observe
        def sample_func(a: int, b: str, secret_val: str, key_param: int) -> str:
            return f"{a}-{b}"

        res = sample_func(10, "hello", secret_val="supersecret", key_param=42)
        self.assertEqual(res, "10-hello")

        # Check spans
        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        span = spans[0]
        self.assertIn("sample_func", span.name)

        # Verify span attributes
        attrs = span.attributes
        self.assertIsNotNone(attrs)
        assert attrs is not None
        self.assertEqual(attrs.get("a"), 10)
        self.assertEqual(attrs.get("b"), "hello")
        # Sensitive key redacted
        self.assertEqual(attrs.get("key_param"), "[REDACTED]")
        # Sensitive val redacted
        self.assertEqual(attrs.get("secret_val"), "[REDACTED]")

    def test_observe_async_function(self) -> None:
        """
        Verify async function span creation and parameter filter/redaction.
        """

        @observe
        async def sample_async_func(x: float, y: bool, auth_token: str) -> bool:
            return y

        res = asyncio.run(sample_async_func(1.5, True, auth_token="token123"))
        self.assertTrue(res)

        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        span = spans[0]
        self.assertIn("sample_async_func", span.name)

        attrs = span.attributes
        self.assertIsNotNone(attrs)
        assert attrs is not None
        self.assertEqual(attrs.get("x"), 1.5)
        self.assertEqual(attrs.get("y"), True)
        self.assertEqual(attrs.get("auth_token"), "[REDACTED]")

    def test_observe_class(self) -> None:
        """
        Verify class decorator observes all public methods.
        """

        @observe
        class SampleClass:
            """Sample class for testing decoration."""

            def __init__(self) -> None:
                pass

            def add(self, x: int, y: int) -> int:
                """Sample public method."""
                return x + y

            async def get_async(self, key_name: str) -> str:
                """Sample async public method."""
                return "value"

        obj = SampleClass()
        self.assertEqual(obj.add(2, 3), 5)

        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        self.assertIn("SampleClass.add", spans[0].name)

        # Reset exporter and check async method
        self.exporter.clear()
        res = asyncio.run(obj.get_async("secretkey"))
        self.assertEqual(res, "value")

        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        self.assertIn("SampleClass.get_async", spans[0].name)
        attrs = spans[0].attributes
        self.assertIsNotNone(attrs)
        assert attrs is not None
        self.assertEqual(attrs.get("key_name"), "[REDACTED]")

    def test_observe_baggage_correlator_id(self) -> None:
        """
        Verify that @observe extracts correlator_id from baggage.
        """
        from opentelemetry import baggage, context

        @observe
        def sample_func() -> str:
            return "done"

        # Set correlator_id in baggage
        ctx = baggage.set_baggage("correlator_id", "my-correlation-id-123")
        token = context.attach(ctx)
        try:
            sample_func()
        finally:
            context.detach(token)

        # Check spans
        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        span = spans[0]
        self.assertIsNotNone(span.attributes)
        assert span.attributes is not None
        self.assertEqual(span.attributes.get("correlator_id"), "my-correlation-id-123")

    def test_observe_async_baggage_correlator_id(self) -> None:
        """
        Verify that @observe extracts correlator_id from async baggage.
        """
        from opentelemetry import baggage, context

        @observe
        async def sample_async_func() -> str:
            return "done"

        # Set correlator_id in baggage
        ctx = baggage.set_baggage("correlator_id", "my-async-correlation-id-456")
        token = context.attach(ctx)
        try:
            asyncio.run(sample_async_func())
        finally:
            context.detach(token)

        # Check spans
        spans = self.exporter.get_finished_spans()
        self.assertEqual(len(spans), 1)
        span = spans[0]
        self.assertIsNotNone(span.attributes)
        assert span.attributes is not None
        self.assertEqual(
            span.attributes.get("correlator_id"), "my-async-correlation-id-456"
        )
