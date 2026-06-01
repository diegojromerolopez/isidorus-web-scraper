package telemetry

import (
	"context"
	"os"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestGetSamplerFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		envType    string
		envArg     string
		descPrefix string
	}{
		{"always_on", "always_on", "", "AlwaysOnSampler"},
		{"always_off", "always_off", "", "AlwaysOffSampler"},
		{"traceidratio", "traceidratio", "0.25", "ErrorAwareSampler"},
		{"parentbased_always_on", "parentbased_always_on", "", "ParentBased"},
		{"parentbased_always_off", "parentbased_always_off", "", "ParentBased"},
		{"parentbased_traceidratio", "parentbased_traceidratio", "0.25", "ParentBased"},
		{"default", "", "", "AlwaysOnSampler"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("OTEL_TRACES_SAMPLER", tt.envType)
			os.Setenv("OTEL_TRACES_SAMPLER_ARG", tt.envArg)
			defer os.Unsetenv("OTEL_TRACES_SAMPLER")
			defer os.Unsetenv("OTEL_TRACES_SAMPLER_ARG")

			sampler := GetSamplerFromEnv()
			if sampler == nil {
				t.Fatal("expected non-nil sampler")
			}
			desc := sampler.Description()
			if !contains(desc, tt.descPrefix) {
				t.Errorf("expected sampler description %q to contain %q", desc, tt.descPrefix)
			}
		})
	}
}

type mockSpanProcessor struct {
	endedSpans []trace.ReadOnlySpan
}

func (p *mockSpanProcessor) OnStart(parent context.Context, s trace.ReadWriteSpan) {}
func (p *mockSpanProcessor) OnEnd(s trace.ReadOnlySpan) {
	p.endedSpans = append(p.endedSpans, s)
}
func (p *mockSpanProcessor) Shutdown(ctx context.Context) error   { return nil }
func (p *mockSpanProcessor) ForceFlush(ctx context.Context) error { return nil }

func TestErrorAwareSpanProcessorAndSampler(t *testing.T) {
	mockProcessor := &mockSpanProcessor{}
	errorAware := NewErrorAwareSpanProcessor(mockProcessor)

	// Set ratio to 0.0 so ratio-based sampling always drops
	sampler := NewErrorAwareSampler(0.0)

	tp := trace.NewTracerProvider(
		trace.WithSampler(sampler),
		trace.WithSpanProcessor(errorAware),
	)
	defer tp.Shutdown(context.Background())

	tracer := tp.Tracer("test-tracer")

	// Case 1: Span succeeds (should NOT be forwarded since ratio is 0.0 and no error)
	ctx1, span1 := tracer.Start(context.Background(), "success-span")
	span1.SetStatus(codes.Ok, "success")
	span1.End()

	// Case 2: Span fails (SHOULD be forwarded because of codes.Error)
	_, span2 := tracer.Start(ctx1, "failed-span")
	span2.SetStatus(codes.Error, "something went wrong")
	span2.End()

	// Verify that only the failed span was exported
	if len(mockProcessor.endedSpans) != 1 {
		t.Fatalf("expected exactly 1 exported span, got %d", len(mockProcessor.endedSpans))
	}

	exportedSpan := mockProcessor.endedSpans[0]
	if exportedSpan.Name() != "failed-span" {
		t.Errorf("expected exported span name to be 'failed-span', got %q", exportedSpan.Name())
	}
}

func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
