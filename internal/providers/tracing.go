// Package providers contains agnostic implementations for external dependencies.
package providers

import (
"context"
"fmt"
"log"
"sync"
"time"

"github.com/AJPalacios/inventory/internal/domain"
"github.com/google/uuid"
)

// TracingConfig holds configuration for DataDog APM tracing provider.
type TracingConfig struct {
	ServiceName string
	Environment string
	AgentHost   string
	AgentPort   int
	SampleRate  float64
	Enabled     bool
}

// NewTracingProvider creates a DataDog APM tracing provider (placeholder).
//
// This is a placeholder implementation that simulates DataDog APM behavior.
// In production, this would use the official dd-trace-go library.
func NewTracingProvider(config TracingConfig) domain.TracingProvider {
	if !config.Enabled {
		return &noopTracingProvider{}
	}
	return NewDataDogTracingProvider(config)
}

// dataDogTracingProvider provides DataDog APM-style tracing (placeholder).
type dataDogTracingProvider struct {
	serviceName string
	environment string
	agentHost   string
	agentPort   int
	sampleRate  float64
	mu          sync.RWMutex
	spans       []*dataDogSpan
}

type dataDogSpan struct {
	traceID       string
	spanID        string
	parentID      string
	serviceName   string
	operationName string
	resourceName  string
	startTime     time.Time
	duration      time.Duration
	tags          map[string]interface{}
	error         error
	logs          []spanLog
	finished      bool
	ctx           context.Context
}

type spanLog struct {
	timestamp time.Time
	event     string
	fields    map[string]interface{}
}

// NewDataDogTracingProvider creates a DataDog APM tracing provider (placeholder).
func NewDataDogTracingProvider(config TracingConfig) domain.TracingProvider {
	log.Printf("[DATADOG APM] Tracer initialized: service=%s env=%s agent=%s:%d sample_rate=%.2f",
config.ServiceName, config.Environment, config.AgentHost, config.AgentPort, config.SampleRate)

	return &dataDogTracingProvider{
		serviceName: config.ServiceName,
		environment: config.Environment,
		agentHost:   config.AgentHost,
		agentPort:   config.AgentPort,
		sampleRate:  config.SampleRate,
		spans:       make([]*dataDogSpan, 0),
	}
}

// StartSpan starts a new tracing span.
func (d *dataDogTracingProvider) StartSpan(ctx context.Context, operationName string) (domain.Span, context.Context) {
	span := &dataDogSpan{
		traceID:       d.getTraceID(ctx),
		spanID:        uuid.New().String(),
		parentID:      d.getParentSpanID(ctx),
		serviceName:   d.serviceName,
		operationName: operationName,
		resourceName:  operationName,
		startTime:     time.Now(),
		tags: map[string]interface{}{
			"env":     d.environment,
			"service": d.serviceName,
		},
		logs:     make([]spanLog, 0),
		finished: false,
		ctx:      ctx,
	}

	d.mu.Lock()
	d.spans = append(d.spans, span)
	d.mu.Unlock()

	// Store span in context
	newCtx := context.WithValue(ctx, spanContextKey, span)
	newCtx = context.WithValue(newCtx, traceIDKey, span.traceID)
	newCtx = context.WithValue(newCtx, parentSpanIDKey, span.spanID)

	log.Printf("[DATADOG APM] Span started: trace_id=%s span_id=%s parent_id=%s operation=%s",
span.traceID, span.spanID, span.parentID, span.operationName)

	return span, newCtx
}

// InjectContext injects tracing context into headers for distributed tracing.
func (d *dataDogTracingProvider) InjectContext(ctx context.Context) map[string]string {
	span := d.getSpanFromContext(ctx)
	if span == nil {
		return make(map[string]string)
	}

	return map[string]string{
		"x-datadog-trace-id":  span.traceID,
		"x-datadog-parent-id": span.spanID,
		"x-datadog-sampling":  fmt.Sprintf("%.2f", d.sampleRate),
	}
}

// ExtractContext extracts tracing context from headers for distributed tracing.
func (d *dataDogTracingProvider) ExtractContext(headers map[string]string) context.Context {
	ctx := context.Background()

	if traceID, ok := headers["x-datadog-trace-id"]; ok {
		ctx = context.WithValue(ctx, traceIDKey, traceID)
	}

	if parentID, ok := headers["x-datadog-parent-id"]; ok {
		ctx = context.WithValue(ctx, parentSpanIDKey, parentID)
	}

	return ctx
}

// Context keys for storing tracing information
type contextKey string

const (
spanContextKey  contextKey = "span"
traceIDKey      contextKey = "trace_id"
parentSpanIDKey contextKey = "parent_span_id"
)

func (d *dataDogTracingProvider) getTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return uuid.New().String()
}

func (d *dataDogTracingProvider) getParentSpanID(ctx context.Context) string {
	if parentID, ok := ctx.Value(parentSpanIDKey).(string); ok {
		return parentID
	}
	if span := d.getSpanFromContext(ctx); span != nil {
		return span.spanID
	}
	return ""
}

func (d *dataDogTracingProvider) getSpanFromContext(ctx context.Context) *dataDogSpan {
	if span, ok := ctx.Value(spanContextKey).(*dataDogSpan); ok {
		return span
	}
	return nil
}

// GetSpans returns all collected spans (for testing/debugging).
func (d *dataDogTracingProvider) GetSpans() []*dataDogSpan {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.spans
}

// Span implementation methods

// SetTag sets a tag on the span.
func (s *dataDogSpan) SetTag(key string, value interface{}) {
	s.tags[key] = value
	log.Printf("[DATADOG APM] Span tag: span_id=%s %s=%v", s.spanID, key, value)
}

// SetError marks the span as errored.
func (s *dataDogSpan) SetError(err error) {
	s.error = err
	s.tags["error"] = true
	s.tags["error.message"] = err.Error()
	s.tags["error.type"] = fmt.Sprintf("%T", err)
	log.Printf("[DATADOG APM] Span error: span_id=%s error=%v", s.spanID, err)
}

// LogEvent logs an event in the span.
func (s *dataDogSpan) LogEvent(event string, fields map[string]interface{}) {
	logEntry := spanLog{
		timestamp: time.Now(),
		event:     event,
		fields:    fields,
	}
	s.logs = append(s.logs, logEntry)
	log.Printf("[DATADOG APM] Span event: span_id=%s event=%s fields=%v", s.spanID, event, fields)
}

// Finish completes the span and sends it to DataDog Agent (placeholder).
func (s *dataDogSpan) Finish() {
	if s.finished {
		return
	}

	s.duration = time.Since(s.startTime)
	s.finished = true

	errorStatus := "ok"
	if s.error != nil {
		errorStatus = "error"
	}

	log.Printf("[DATADOG APM] Span finished: trace_id=%s span_id=%s operation=%s duration=%dms status=%s tags=%v",
s.traceID, s.spanID, s.operationName, s.duration.Milliseconds(), errorStatus, s.tags)

	// In a real implementation, this would serialize and send the span to DataDog Agent
	// via HTTP POST to http://\{agent_host\}:\{agent_port\}/v0.4/traces
}

// Context returns the span's context.
func (s *dataDogSpan) Context() context.Context {
return s.ctx
}

// noopTracingProvider provides a no-op tracing implementation.
type noopTracingProvider struct{}

type noopSpan struct {
ctx context.Context
}

func (n *noopTracingProvider) StartSpan(ctx context.Context, operationName string) (domain.Span, context.Context) {
return &noopSpan{ctx: ctx}, ctx
}

func (n *noopTracingProvider) InjectContext(ctx context.Context) map[string]string {
return make(map[string]string)
}

func (n *noopTracingProvider) ExtractContext(headers map[string]string) context.Context {
return context.Background()
}

func (s *noopSpan) SetTag(key string, value interface{})                  {}
func (s *noopSpan) SetError(err error)                                    {}
func (s *noopSpan) LogEvent(event string, fields map[string]interface{}) {}
func (s *noopSpan) Finish()                                               {}
func (s *noopSpan) Context() context.Context                              { return s.ctx }
