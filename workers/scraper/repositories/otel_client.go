package repositories

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TelemetryClient interface {
	StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, TelemetrySpan)
	Shutdown(ctx context.Context) error
}

type TelemetrySpan interface {
	End()
	SetAttribute(key string, value interface{})
	RecordError(err error)
	SetStatus(code string, description string)
}

type SpanOption func(*spanOptions)

type spanOptions struct {
	attributes map[string]interface{}
}

func WithAttribute(key string, value interface{}) SpanOption {
	return func(o *spanOptions) {
		if o.attributes == nil {
			o.attributes = make(map[string]interface{})
		}
		o.attributes[key] = value
	}
}

type otelTelemetryClient struct {
	tp     *trace.TracerProvider
	tracer oteltrace.Tracer
}

func NewTelemetryClient(tp *trace.TracerProvider, serviceName string) TelemetryClient {
	return &otelTelemetryClient{
		tp:     tp,
		tracer: tp.Tracer(serviceName),
	}
}

func (c *otelTelemetryClient) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, TelemetrySpan) {
	so := &spanOptions{}
	for _, opt := range opts {
		opt(so)
	}

	var otelOpts []oteltrace.SpanStartOption
	if len(so.attributes) > 0 {
		var attrs []attribute.KeyValue
		for k, v := range so.attributes {
			attrs = append(attrs, mapToAttribute(k, v))
		}
		otelOpts = append(otelOpts, oteltrace.WithAttributes(attrs...))
	}

	ctx, span := c.tracer.Start(ctx, name, otelOpts...)
	return ctx, &otelTelemetrySpan{span: span}
}

func (c *otelTelemetryClient) Shutdown(ctx context.Context) error {
	if c.tp != nil {
		return c.tp.Shutdown(ctx)
	}
	return nil
}

type otelTelemetrySpan struct {
	span oteltrace.Span
}

func (s *otelTelemetrySpan) End() {
	s.span.End()
}

func (s *otelTelemetrySpan) SetAttribute(key string, value interface{}) {
	s.span.SetAttributes(mapToAttribute(key, value))
}

func (s *otelTelemetrySpan) RecordError(err error) {
	s.span.RecordError(err)
}

func (s *otelTelemetrySpan) SetStatus(code string, description string) {
	switch strings.ToLower(code) {
	case "error":
		s.span.SetStatus(codes.Error, description)
	case "ok":
		s.span.SetStatus(codes.Ok, description)
	default:
		s.span.SetStatus(codes.Unset, description)
	}
}

// No-op client for compatibility and easy testing
type noopTelemetryClient struct{}

func NewNoopTelemetryClient() TelemetryClient {
	return &noopTelemetryClient{}
}

func (c *noopTelemetryClient) StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, TelemetrySpan) {
	return ctx, &noopTelemetrySpan{}
}

func (c *noopTelemetryClient) Shutdown(ctx context.Context) error {
	return nil
}

type noopTelemetrySpan struct{}

func (s *noopTelemetrySpan) End()                                       {}
func (s *noopTelemetrySpan) SetAttribute(key string, value interface{}) {}
func (s *noopTelemetrySpan) RecordError(err error)                      {}
func (s *noopTelemetrySpan) SetStatus(code string, description string)  {}

func isSensitive(s string) bool {
	s = strings.ToLower(s)
	for _, term := range []string{"secret", "token", "key", "password", "auth"} {
		if strings.Contains(s, term) {
			return true
		}
	}
	return false
}

func redactValue(key string, val interface{}) interface{} {
	if isSensitive(key) {
		return "[REDACTED]"
	}
	if strVal, ok := val.(string); ok {
		if isSensitive(strVal) {
			return "[REDACTED]"
		}
	}
	return val
}

func mapToAttribute(key string, val interface{}) attribute.KeyValue {
	val = redactValue(key, val)

	switch v := val.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int64(key, int64(v))
	case int8:
		return attribute.Int64(key, int64(v))
	case int16:
		return attribute.Int64(key, int64(v))
	case int32:
		return attribute.Int64(key, int64(v))
	case int64:
		return attribute.Int64(key, v)
	case uint:
		return attribute.Int64(key, int64(v))
	case uint8:
		return attribute.Int64(key, int64(v))
	case uint16:
		return attribute.Int64(key, int64(v))
	case uint32:
		return attribute.Int64(key, int64(v))
	case uint64:
		return attribute.Int64(key, int64(v))
	case float32:
		return attribute.Float64(key, float64(v))
	case float64:
		return attribute.Float64(key, v)
	default:
		return attribute.String(key, fmt.Sprintf("%v", v))
	}
}
