package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTelemetryClient_Spans(t *testing.T) {
	exporter := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(exporter))
	defer exporter.Reset()

	client := NewTelemetryClient(tp, "test-service")

	ctx := context.Background()
	_, span := client.StartSpan(ctx, "test-span",
		WithAttribute("normal-string", "hello"),
		WithAttribute("normal-int", 42),
		WithAttribute("sensitive-password", "my-secret-pass"),
	)
	span.SetAttribute("another-sensitive", "some_secret_token")
	span.SetAttribute("bool-val", true)
	span.SetAttribute("float-val", 3.14)
	span.SetStatus("error", "some failure occurred")
	span.RecordError(errors.New("test error"))
	span.End()

	spans := exporter.Ended()
	assert.Len(t, spans, 1)
	capturedSpan := spans[0]
	assert.Equal(t, "test-span", capturedSpan.Name())

	attrs := capturedSpan.Attributes()
	var normalStrOk, normalIntOk, boolOk, floatOk, sensitivePassRedacted, sensitiveTokenRedacted bool

	for _, attr := range attrs {
		switch attr.Key {
		case "normal-string":
			assert.Equal(t, "hello", attr.Value.AsString())
			normalStrOk = true
		case "normal-int":
			assert.Equal(t, int64(42), attr.Value.AsInt64())
			normalIntOk = true
		case "bool-val":
			assert.Equal(t, true, attr.Value.AsBool())
			boolOk = true
		case "float-val":
			assert.Equal(t, 3.14, attr.Value.AsFloat64())
			floatOk = true
		case "sensitive-password":
			assert.Equal(t, "[REDACTED]", attr.Value.AsString())
			sensitivePassRedacted = true
		case "another-sensitive":
			assert.Equal(t, "[REDACTED]", attr.Value.AsString())
			sensitiveTokenRedacted = true
		}
	}

	assert.True(t, normalStrOk)
	assert.True(t, normalIntOk)
	assert.True(t, boolOk)
	assert.True(t, floatOk)
	assert.True(t, sensitivePassRedacted)
	assert.True(t, sensitiveTokenRedacted)
}

func TestTelemetryClient_BaggageCorrelationID(t *testing.T) {
	exporter := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(exporter))
	defer exporter.Reset()

	client := NewTelemetryClient(tp, "test-service")

	// Create context with correlator_id in baggage
	ctx := context.Background()
	m, err := baggage.NewMember("correlator_id", "my-unique-correlator-id")
	assert.NoError(t, err)
	b, err := baggage.New(m)
	assert.NoError(t, err)
	ctx = baggage.ContextWithBaggage(ctx, b)

	_, span := client.StartSpan(ctx, "test-span-with-baggage")
	span.End()

	spans := exporter.Ended()
	assert.Len(t, spans, 1)
	capturedSpan := spans[0]
	assert.Equal(t, "test-span-with-baggage", capturedSpan.Name())

	var foundCorrelationID bool
	for _, attr := range capturedSpan.Attributes() {
		if attr.Key == "correlator_id" {
			assert.Equal(t, "my-unique-correlator-id", attr.Value.AsString())
			foundCorrelationID = true
		}
	}
	assert.True(t, foundCorrelationID, "should have found correlator_id attribute")
}

func TestTelemetryClient_ContextPropagation(t *testing.T) {
	exporter := tracetest.NewSpanRecorder()
	tp := trace.NewTracerProvider(trace.WithSpanProcessor(exporter))
	defer exporter.Reset()

	client := NewTelemetryClient(tp, "test-service")

	// 1. Create a parent span
	ctx := context.Background()
	parentCtx, parentSpan := client.StartSpan(ctx, "parent-span")

	// 2. Inject trace context into a carrier (simulating SQS payload injection)
	traceMap := make(map[string]string)
	otel.GetTextMapPropagator().Inject(parentCtx, propagation.MapCarrier(traceMap))
	parentSpan.End()

	// 3. Extract trace context from the carrier (simulating SQS payload extraction)
	extractedCtx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier(traceMap))

	// 4. Start a child span under the extracted context
	_, childSpan := client.StartSpan(extractedCtx, "child-span")
	childSpan.End()

	spans := exporter.Ended()
	assert.Len(t, spans, 2)

	var pSpan, cSpan trace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == "parent-span" {
			pSpan = s
		} else if s.Name() == "child-span" {
			cSpan = s
		}
	}

	assert.NotNil(t, pSpan)
	assert.NotNil(t, cSpan)

	// Verify parent-child relationship across context propagation
	assert.Equal(t, pSpan.SpanContext().TraceID(), cSpan.SpanContext().TraceID(), "trace IDs must match")
	assert.Equal(t, pSpan.SpanContext().SpanID(), cSpan.Parent().SpanID(), "child's parent span ID must match parent's span ID")
}
