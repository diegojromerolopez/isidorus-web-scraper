package telemetry

import (
	"context"
	"os"
	"strconv"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
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
