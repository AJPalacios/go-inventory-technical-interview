package providers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsProviderFactory tests the metrics provider factory
func TestMetricsProviderFactory(t *testing.T) {
	// Test memory provider
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
		Labels:    map[string]string{"env": "test"},
	}

	provider := NewMetricsProvider(config)
	assert.NotNil(t, provider)

	// Test DataDog provider
	config.Provider = MetricsProviderDataDog
	provider = NewMetricsProvider(config)
	assert.NotNil(t, provider)

	// Test unknown provider fallback
	config.Provider = "unknown"
	provider = NewMetricsProvider(config)
	assert.NotNil(t, provider)
}

func TestMemoryMetricsProvider(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
		Labels:    map[string]string{"env": "test"},
	}

	provider := NewMemoryMetricsProvider(config)
	require.NotNil(t, provider)

	memProvider := provider.(*memoryMetricsProvider)

	// Test IncrementCounter
	provider.IncrementCounter("test_counter", map[string]string{"method": "GET"})
	provider.IncrementCounter("test_counter", map[string]string{"method": "GET"})

	metrics := memProvider.GetMetrics()
	counters := metrics["counters"].(map[string]float64)
	assert.Greater(t, len(counters), 0)

	// Test RecordDuration
	provider.RecordDuration("test_duration", time.Millisecond*100, map[string]string{"operation": "reserve"})

	histograms := metrics["histograms"].(map[string][]float64)
	assert.Greater(t, len(histograms), 0)

	// Test Reset
	memProvider.Reset()
	metricsAfterReset := memProvider.GetMetrics()
	countersAfterReset := metricsAfterReset["counters"].(map[string]float64)
	assert.Equal(t, 0, len(countersAfterReset))
}

func TestDataDogMetricsProvider(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderDataDog,
		Namespace: "inventory",
		Address:   "localhost:8125",
		Labels:    map[string]string{"service": "inventory", "version": "1.0"},
	}

	provider := NewDataDogMetricsProvider(config)
	require.NotNil(t, provider)

	ddProvider := provider.(*dataDogMetricsProvider)

	// Test IncrementCounter
	provider.IncrementCounter("requests_total", map[string]string{"method": "POST"})

	metrics := ddProvider.GetMetrics()
	assert.Len(t, metrics, 1)
	assert.Equal(t, "counter", metrics[0].Type)
	assert.Equal(t, "inventory.requests_total", metrics[0].MetricName)

	// Test RecordDuration
	provider.RecordDuration("request_duration", time.Millisecond*250, map[string]string{"endpoint": "/reserve"})

	metrics = ddProvider.GetMetrics()
	assert.Len(t, metrics, 2)
	assert.Equal(t, "histogram", metrics[1].Type)
	assert.Equal(t, 250.0, metrics[1].Value)
}

func TestMemoryMetricsProvider_IncrementCounterBy(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
	}

	provider := NewMemoryMetricsProvider(config)
	memProvider := provider.(*memoryMetricsProvider)

	// Test IncrementCounterBy
	memProvider.IncrementCounterBy("test_counter", 5.0, map[string]string{"label": "value"})
	memProvider.IncrementCounterBy("test_counter", 3.0, map[string]string{"label": "value"})

	metrics := memProvider.GetMetrics()
	counters := metrics["counters"].(map[string]float64)

	// Should have one entry with value 8.0
	var found bool
	for _, value := range counters {
		if value == 8.0 {
			found = true
			break
		}
	}
	assert.True(t, found, "Counter should have value 8.0")
}

func TestMemoryMetricsProvider_Gauges(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
	}

	provider := NewMemoryMetricsProvider(config)
	memProvider := provider.(*memoryMetricsProvider)

	// Test SetGauge
	memProvider.SetGauge("active_connections", 42.0, map[string]string{"server": "web1"})

	// Test AddGauge
	memProvider.AddGauge("active_connections", 8.0, map[string]string{"server": "web1"})

	metrics := memProvider.GetMetrics()
	gauges := metrics["gauges"].(map[string]float64)

	var found bool
	for _, value := range gauges {
		if value == 50.0 {
			found = true
			break
		}
	}
	assert.True(t, found, "Gauge should have value 50.0")
}

func TestMemoryMetricsProvider_Histograms(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
	}

	provider := NewMemoryMetricsProvider(config)
	memProvider := provider.(*memoryMetricsProvider)

	// Test RecordHistogram
	memProvider.RecordHistogram("response_time", 123.45, map[string]string{"endpoint": "/api"})
	memProvider.RecordHistogram("response_time", 67.89, map[string]string{"endpoint": "/api"})

	metrics := memProvider.GetMetrics()
	histograms := metrics["histograms"].(map[string][]float64)

	var found bool
	for _, values := range histograms {
		if len(values) == 2 && values[0] == 123.45 && values[1] == 67.89 {
			found = true
			break
		}
	}
	assert.True(t, found, "Histogram should have both values")
}

func TestDataDogMetricsProvider_AllMethods(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderDataDog,
		Namespace: "test",
		Labels:    map[string]string{"env": "prod"},
	}

	provider := NewDataDogMetricsProvider(config)
	ddProvider := provider.(*dataDogMetricsProvider)

	// Test all methods
	ddProvider.IncrementCounterBy("counter_by", 10.0, map[string]string{"type": "test"})
	ddProvider.SetGauge("gauge_set", 100.0, map[string]string{"type": "test"})
	ddProvider.AddGauge("gauge_add", 25.0, map[string]string{"type": "test"})
	ddProvider.RecordHistogram("histogram", 75.5, map[string]string{"type": "test"})

	metrics := ddProvider.GetMetrics()
	assert.Len(t, metrics, 4)

	// Check metric names have namespace
	for _, metric := range metrics {
		assert.Contains(t, metric.MetricName, "test.")
		assert.NotEmpty(t, metric.Tags)
		assert.Contains(t, metric.Tags, "env:prod")
		assert.Contains(t, metric.Tags, "type:test")
	}
}

func TestBuildKey(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test",
		Labels:    map[string]string{"env": "dev"},
	}

	provider := NewMemoryMetricsProvider(config)
	memProvider := provider.(*memoryMetricsProvider)

	key := memProvider.buildKey("metric_name", map[string]string{"label1": "value1", "label2": "value2"})

	assert.Contains(t, key, "test_metric_name")
	assert.Contains(t, key, "env=dev")
	assert.Contains(t, key, "label1=value1")
	assert.Contains(t, key, "label2=value2")
}

func TestDataDogBuildTags(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderDataDog,
		Namespace: "test",
		Labels:    map[string]string{"service": "inventory"},
	}

	provider := NewDataDogMetricsProvider(config)
	ddProvider := provider.(*dataDogMetricsProvider)

	tags := ddProvider.buildTags(map[string]string{"method": "GET", "status": "200"})

	assert.Contains(t, tags, "service:inventory")
	assert.Contains(t, tags, "method:GET")
	assert.Contains(t, tags, "status:200")
	assert.Len(t, tags, 3)
}

func TestMetricsConfig(t *testing.T) {
	config := MetricsConfig{
		Provider:  MetricsProviderMemory,
		Namespace: "test_namespace",
		Address:   "localhost:8125",
		Labels:    map[string]string{"app": "inventory"},
	}

	assert.Equal(t, MetricsProviderMemory, config.Provider)
	assert.Equal(t, "test_namespace", config.Namespace)
	assert.Equal(t, "localhost:8125", config.Address)
	assert.Equal(t, "inventory", config.Labels["app"])
}

func TestMetricsProviderConstants(t *testing.T) {
	assert.Equal(t, MetricsProviderType("memory"), MetricsProviderMemory)
	assert.Equal(t, MetricsProviderType("datadog"), MetricsProviderDataDog)
}
