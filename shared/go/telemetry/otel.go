package telemetry

import (
	"context"
	"log"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// errorAwareSampler wraps another sampler and returns RecordOnly
// instead of Drop when the delegate returns Drop. This ensures we can
// capture and export failed spans downstream even when sampling is probabilistic.
type errorAwareSampler struct {
	ratioSampler trace.Sampler
}

func NewErrorAwareSampler(ratio float64) trace.Sampler {
	return &errorAwareSampler{
		ratioSampler: trace.TraceIDRatioBased(ratio),
	}
}

func (s *errorAwareSampler) ShouldSample(p trace.SamplingParameters) trace.SamplingResult {
	result := s.ratioSampler.ShouldSample(p)
	if result.Decision == trace.RecordAndSample {
		return result
	}
	return trace.SamplingResult{
		Decision:   trace.RecordOnly,
		Attributes: result.Attributes,
		Tracestate: result.Tracestate,
	}
}

func (s *errorAwareSampler) Description() string {
	return "ErrorAwareSampler"
}

// errorAwareSpanProcessor wraps another span processor and only forwards spans
// that were sampled OR had a failure status (codes.Error).
type errorAwareSpanProcessor struct {
	delegate trace.SpanProcessor
}

func NewErrorAwareSpanProcessor(delegate trace.SpanProcessor) trace.SpanProcessor {
	return &errorAwareSpanProcessor{delegate: delegate}
}

func (p *errorAwareSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {
	p.delegate.OnStart(parent, s)
}

func (p *errorAwareSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	if s.SpanContext().IsSampled() || s.Status().Code == codes.Error {
		p.delegate.OnEnd(s)
	}
}

func (p *errorAwareSpanProcessor) Shutdown(ctx context.Context) error {
	return p.delegate.Shutdown(ctx)
}

func (p *errorAwareSpanProcessor) ForceFlush(ctx context.Context) error {
	return p.delegate.ForceFlush(ctx)
}

// GetSamplerFromEnv reads the OTEL_TRACES_SAMPLER and OTEL_TRACES_SAMPLER_ARG
// environment variables and returns the corresponding trace.Sampler.
// Defaults to trace.AlwaysSample() if not set or invalid.
func GetSamplerFromEnv() trace.Sampler {
	samplerType := os.Getenv("OTEL_TRACES_SAMPLER")
	samplerArgStr := os.Getenv("OTEL_TRACES_SAMPLER_ARG")

	var ratio float64 = 1.0
	if samplerArgStr != "" {
		if r, err := strconv.ParseFloat(samplerArgStr, 64); err == nil {
			ratio = r
		}
	}

	switch samplerType {
	case "always_on":
		return trace.AlwaysSample()
	case "always_off":
		return trace.NeverSample()
	case "traceidratio":
		return NewErrorAwareSampler(ratio)
	case "parentbased_always_on":
		return trace.ParentBased(trace.AlwaysSample())
	case "parentbased_always_off":
		return trace.ParentBased(trace.NeverSample())
	case "parentbased_traceidratio":
		return trace.ParentBased(NewErrorAwareSampler(ratio))
	default:
		// Default to always_on (AlwaysSample) to preserve existing behavior
		return trace.AlwaysSample()
	}
}

// InitTelemetry initializes standard OpenTelemetry tracing with OTLP exporting
// and the environment-configured trace sampler.
func InitTelemetry(ctx context.Context, serviceName string) (*trace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	sampler := GetSamplerFromEnv()
	tpOpts := []trace.TracerProviderOption{
		trace.WithSampler(sampler),
		trace.WithResource(res),
	}

	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint != "" {
		// Create OTLP gRPC trace exporter.
		// Note: otlptracegrpc.New respects standard OTel env variables.
		exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure())
		if err == nil {
			batchProcessor := trace.NewBatchSpanProcessor(exporter)
			tpOpts = append(tpOpts, trace.WithSpanProcessor(NewErrorAwareSpanProcessor(batchProcessor)))
		}

		// Create OTLP gRPC metrics exporter.
		metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure())
		if err == nil {
			mp := sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(res),
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
			)
			otel.SetMeterProvider(mp)
		}
	}

	tp := trace.NewTracerProvider(tpOpts...)
	otel.SetTracerProvider(tp)

	// Set global W3C textmap propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}

// GetMeter returns the standard OpenTelemetry meter for registering metrics.
func GetMeter(serviceName string) metric.Meter {
	return otel.GetMeterProvider().Meter(serviceName)
}

// LogWithTrace prints a log line automatically prefixed with [trace_id][span_id] if tracing context is valid.
func LogWithTrace(ctx context.Context, format string, v ...interface{}) {
	spanCtx := oteltrace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		prefix := "[" + spanCtx.TraceID().String() + "][" + spanCtx.SpanID().String() + "] "
		log.Printf(prefix+format, v...)
	} else {
		log.Printf(format, v...)
	}
}
