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
	MetricsProviderMemory     MetricsProviderType = "memory"
	MetricsProviderPrometheus MetricsProviderType = "prometheus"
	MetricsProviderStatsD     MetricsProviderType = "statsd"
	MetricsProviderDataDog    MetricsProviderType = "datadog"
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
// This factory function allows switching between different metrics
// providers without changing the business logic code.
func NewMetricsProvider(config MetricsConfig) domain.MetricsProvider {
	switch config.Provider {
	case MetricsProviderMemory:
		return NewMemoryMetricsProvider(config)
	case MetricsProviderPrometheus:
		// return NewPrometheusMetricsProvider(config)
		log.Printf("Prometheus provider not implemented, falling back to memory")
		return NewMemoryMetricsProvider(config)
	case MetricsProviderStatsD:
		// return NewStatsDMetricsProvider(config)
		log.Printf("StatsD provider not implemented, falling back to memory")
		return NewMemoryMetricsProvider(config)
	case MetricsProviderDataDog:
		// return NewDataDogMetricsProvider(config)
		log.Printf("DataDog provider not implemented, falling back to memory")
		return NewMemoryMetricsProvider(config)
	default:
		log.Printf("Unknown metrics provider %s, falling back to memory", config.Provider)
		return NewMemoryMetricsProvider(config)
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
