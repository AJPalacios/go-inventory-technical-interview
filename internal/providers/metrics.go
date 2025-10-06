// Package providers contains agnostic implementations for external dependencies.
//
// This package provides pluggable implementations that can be swapped
// without changing business logic, following the Strategy pattern.
package providers

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/AJPalacios/inventory/internal/domain"
)

// MetricsProviderType defines available metrics provider types.
type MetricsProviderType string

const (
	MetricsProviderMemory  MetricsProviderType = "memory"
	MetricsProviderDataDog MetricsProviderType = "datadog"
)

// MetricsConfig holds configuration for metrics provider.
type MetricsConfig struct {
	Provider  MetricsProviderType
	Namespace string
	Address   string
	Labels    map[string]string
}

// NewMetricsProvider creates a metrics provider based on configuration.
//
// Supports DataDog metrics (placeholder) and in-memory for testing.
func NewMetricsProvider(config MetricsConfig) domain.MetricsProvider {
	switch config.Provider {
	case MetricsProviderDataDog:
		return NewDataDogMetricsProvider(config)
	case MetricsProviderMemory:
		return NewMemoryMetricsProvider(config)
	default:
		log.Printf("Unknown metrics provider %s, falling back to DataDog", config.Provider)
		return NewDataDogMetricsProvider(config)
	}
}

// memoryMetricsProvider provides in-memory metrics collection for testing/development.
//
// This implementation stores metrics in memory and provides basic
// functionality for development and testing purposes.
type memoryMetricsProvider struct {
	namespace  string
	labels     map[string]string
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
	mu         sync.RWMutex
}

// NewMemoryMetricsProvider creates an in-memory metrics provider.
func NewMemoryMetricsProvider(config MetricsConfig) domain.MetricsProvider {
	return &memoryMetricsProvider{
		namespace:  config.Namespace,
		labels:     config.Labels,
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
	}
}

// IncrementCounter increments a counter metric by 1.
func (m *memoryMetricsProvider) IncrementCounter(name string, labels map[string]string) {
	m.IncrementCounterBy(name, 1, labels)
}

// IncrementCounterBy increments a counter metric by the specified value.
func (m *memoryMetricsProvider) IncrementCounterBy(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, labels)
	m.counters[key] += value
}

// SetGauge sets a gauge metric to the specified value.
func (m *memoryMetricsProvider) SetGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, labels)
	m.gauges[key] = value
}

// AddGauge adds to a gauge metric value.
func (m *memoryMetricsProvider) AddGauge(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, labels)
	m.gauges[key] += value
}

// RecordHistogram records a histogram metric value.
func (m *memoryMetricsProvider) RecordHistogram(name string, value float64, labels map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.buildKey(name, labels)
	m.histograms[key] = append(m.histograms[key], value)
}

// RecordDuration records a duration metric.
func (m *memoryMetricsProvider) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	m.RecordHistogram(name, float64(duration.Nanoseconds())/1e6, labels) // Convert to milliseconds
}

// buildKey builds a unique key for a metric with labels.
func (m *memoryMetricsProvider) buildKey(name string, labels map[string]string) string {
	key := fmt.Sprintf("%s_%s", m.namespace, name)

	// Add global labels
	for k, v := range m.labels {
		key += fmt.Sprintf("_%s=%s", k, v)
	}

	// Add specific labels
	for k, v := range labels {
		key += fmt.Sprintf("_%s=%s", k, v)
	}

	return key
}

// GetMetrics returns all collected metrics (for testing/debugging).
func (m *memoryMetricsProvider) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"counters":   m.counters,
		"gauges":     m.gauges,
		"histograms": m.histograms,
	}
}

// Reset clears all metrics (for testing).
func (m *memoryMetricsProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]float64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
}

// dataDogMetricsProvider provides DataDog-style metrics collection (placeholder).
//
// This is a placeholder implementation that simulates DataDog metrics
// for demonstration purposes. In production, this would use the official
// DataDog client library.
type dataDogMetricsProvider struct {
	namespace string
	labels    map[string]string
	address   string
	mu        sync.RWMutex
	metrics   []dataDogMetric
}

type dataDogMetric struct {
	Timestamp  time.Time
	MetricName string
	Value      float64
	Type       string // "counter", "gauge", "histogram"
	Tags       []string
}

// NewDataDogMetricsProvider creates a DataDog metrics provider (placeholder).
func NewDataDogMetricsProvider(config MetricsConfig) domain.MetricsProvider {
	return &dataDogMetricsProvider{
		namespace: config.Namespace,
		labels:    config.Labels,
		address:   config.Address,
		metrics:   make([]dataDogMetric, 0),
	}
}

// IncrementCounter increments a counter metric by 1.
func (d *dataDogMetricsProvider) IncrementCounter(name string, labels map[string]string) {
	d.IncrementCounterBy(name, 1, labels)
}

// IncrementCounterBy increments a counter metric by the specified value.
func (d *dataDogMetricsProvider) IncrementCounterBy(name string, value float64, labels map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	metric := dataDogMetric{
		Timestamp:  time.Now(),
		MetricName: d.buildMetricName(name),
		Value:      value,
		Type:       "counter",
		Tags:       d.buildTags(labels),
	}

	d.metrics = append(d.metrics, metric)
	d.logMetric(metric)
}

// SetGauge sets a gauge metric to the specified value.
func (d *dataDogMetricsProvider) SetGauge(name string, value float64, labels map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	metric := dataDogMetric{
		Timestamp:  time.Now(),
		MetricName: d.buildMetricName(name),
		Value:      value,
		Type:       "gauge",
		Tags:       d.buildTags(labels),
	}

	d.metrics = append(d.metrics, metric)
	d.logMetric(metric)
}

// AddGauge adds to a gauge metric value.
func (d *dataDogMetricsProvider) AddGauge(name string, value float64, labels map[string]string) {
	// In DataDog, we just set the gauge with the incremented value
	d.SetGauge(name, value, labels)
}

// RecordHistogram records a histogram metric value.
func (d *dataDogMetricsProvider) RecordHistogram(name string, value float64, labels map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	metric := dataDogMetric{
		Timestamp:  time.Now(),
		MetricName: d.buildMetricName(name),
		Value:      value,
		Type:       "histogram",
		Tags:       d.buildTags(labels),
	}

	d.metrics = append(d.metrics, metric)
	d.logMetric(metric)
}

// RecordDuration records a duration metric.
func (d *dataDogMetricsProvider) RecordDuration(name string, duration time.Duration, labels map[string]string) {
	// Convert to milliseconds for DataDog
	d.RecordHistogram(name, float64(duration.Milliseconds()), labels)
}

// buildMetricName builds a DataDog-style metric name with namespace.
func (d *dataDogMetricsProvider) buildMetricName(name string) string {
	if d.namespace != "" {
		return fmt.Sprintf("%s.%s", d.namespace, name)
	}
	return name
}

// buildTags builds DataDog-style tags from labels.
func (d *dataDogMetricsProvider) buildTags(labels map[string]string) []string {
	tags := make([]string, 0)

	// Add global labels as tags
	for k, v := range d.labels {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}

	// Add specific labels as tags
	for k, v := range labels {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}

	return tags
}

// logMetric logs the metric in DataDog format (placeholder for actual API call).
func (d *dataDogMetricsProvider) logMetric(metric dataDogMetric) {
	// In a real implementation, this would send to DataDog API
	// For now, we just log it in DataDog-compatible format
	log.Printf("[DATADOG] %s type=%s value=%.2f tags=%v timestamp=%d",
		metric.MetricName,
		metric.Type,
		metric.Value,
		metric.Tags,
		metric.Timestamp.Unix(),
	)
}

// GetMetrics returns all collected metrics (for testing/debugging).
func (d *dataDogMetricsProvider) GetMetrics() []dataDogMetric {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return d.metrics
}
