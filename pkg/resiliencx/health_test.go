package resiliencx

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHealthCheckHealthy(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{Name: "resiliencx"}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(context.Background())
	if status.Status != HealthHealthy {
		t.Fatalf("expected healthy status, got %q", status.Status)
	}
	if status.Name != "resiliencx" {
		t.Fatalf("expected resiliencx health name, got %q", status.Name)
	}
	if status.LatencyMs < 0 {
		t.Fatalf("expected non-negative latency, got %d", status.LatencyMs)
	}
	if !metrics.hasGauge(MetricClientHealthStatus) {
		t.Fatalf("expected health status gauge, got %#v", metrics.gauges)
	}
	if !metrics.hasHistogram(MetricClientHealthLatencyMS) {
		t.Fatalf("expected health latency histogram, got %#v", metrics.histograms)
	}
}

func TestHealthCheckClosedClientUnhealthy(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "resiliencx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("close client: %v", err)
	}

	status := client.HealthCheck(context.Background())
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
}

func TestHealthCheckCanceledContextUnhealthy(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "resiliencx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	status := client.HealthCheck(ctx)
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if !strings.Contains(status.Message, context.Canceled.Error()) {
		t.Fatalf("expected canceled message, got %q", status.Message)
	}
}

func TestHealthCheckNilContextUnhealthy(t *testing.T) {
	client, err := New(context.Background(), Config{Name: "resiliencx"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(nil) //nolint:staticcheck // verifies the defensive nil-context branch.
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if status.Message != "context is required" {
		t.Fatalf("expected nil context message, got %q", status.Message)
	}
}

func TestHealthCheckDeadlineBelowTimeoutDegraded(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{
		Name:    "resiliencx",
		Timeout: time.Hour,
	}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := client.HealthCheck(ctx)
	if status.Status != HealthDegraded {
		t.Fatalf("expected degraded status, got %q", status.Status)
	}
	if status.Message != "context deadline is shorter than client timeout" {
		t.Fatalf("expected degraded message, got %q", status.Message)
	}
	if status.Metadata["reason"] != "deadline_below_timeout" {
		t.Fatalf("expected degraded reason metadata, got %#v", status.Metadata)
	}
	if status.Metadata["timeout"] != time.Hour.String() {
		t.Fatalf("expected timeout metadata, got %#v", status.Metadata)
	}

	payload, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal degraded health status: %v", err)
	}
	encoded := string(payload)
	for _, field := range []string{"name", "status", "checked_at", "latency_ms", "metadata"} {
		if !strings.Contains(encoded, `"`+field+`"`) {
			t.Fatalf("expected JSON field %q in %s", field, encoded)
		}
	}
	if !strings.Contains(encoded, `"status":"degraded"`) {
		t.Fatalf("expected degraded JSON status, got %s", encoded)
	}
	if strings.Contains(encoded, "CheckedAt") || strings.Contains(encoded, "LatencyMs") {
		t.Fatalf("expected snake_case JSON fields, got %s", encoded)
	}

	labels := map[string]string{
		"name":   "resiliencx",
		"status": string(HealthDegraded),
	}
	if !metrics.gaugeWithLabels(MetricClientHealthStatus, 0, labels) {
		t.Fatalf("expected degraded health status gauge, got %#v", metrics.gauges)
	}
	if !metrics.histogramWithLabels(MetricClientHealthLatencyMS, labels) {
		t.Fatalf("expected degraded health latency histogram, got %#v", metrics.histograms)
	}
}

func TestHealthCheckTimeoutWithoutDeadlineHealthy(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{
		Name:    "resiliencx",
		Timeout: time.Minute,
	}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(context.Background())
	if status.Status != HealthHealthy {
		t.Fatalf("expected healthy status, got %q", status.Status)
	}
	if status.Metadata != nil {
		t.Fatalf("expected no health metadata, got %#v", status.Metadata)
	}

	labels := map[string]string{
		"name":   "resiliencx",
		"status": string(HealthHealthy),
	}
	if !metrics.gaugeWithLabels(MetricClientHealthStatus, 1, labels) {
		t.Fatalf("expected healthy health status gauge, got %#v", metrics.gauges)
	}
	if !metrics.histogramWithLabels(MetricClientHealthLatencyMS, labels) {
		t.Fatalf("expected healthy health latency histogram, got %#v", metrics.histograms)
	}
}

func TestHealthCheckDeadlineAboveTimeoutHealthy(t *testing.T) {
	client, err := New(context.Background(), Config{
		Name:    "resiliencx",
		Timeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	status := client.HealthCheck(ctx)
	if status.Status != HealthHealthy {
		t.Fatalf("expected healthy status, got %q", status.Status)
	}
	if status.Metadata["reason"] == "deadline_below_timeout" {
		t.Fatalf("expected no degraded reason metadata, got %#v", status.Metadata)
	}
}

func TestHealthCheckNilClientUnhealthy(t *testing.T) {
	var client *Client

	status := client.HealthCheck(context.Background())
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if status.Name != "resiliencx" {
		t.Fatalf("expected fallback health name, got %q", status.Name)
	}
}

func TestHealthCheckElapsedDeadlineWithoutContextErrorUnhealthy(t *testing.T) {
	metrics := &recordingMetrics{}
	client, err := New(context.Background(), Config{
		Name:    "resiliencx",
		Timeout: time.Hour,
	}, WithMetrics(metrics))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(deadlineOnlyContext{
		Context:  context.Background(),
		deadline: time.Now().Add(-time.Second),
	})
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if status.Message != context.DeadlineExceeded.Error() {
		t.Fatalf("expected deadline message, got %q", status.Message)
	}

	labels := map[string]string{
		"name":   "resiliencx",
		"status": string(HealthUnhealthy),
	}
	if !metrics.gaugeWithLabels(MetricClientHealthStatus, 0, labels) {
		t.Fatalf("expected unhealthy health status gauge, got %#v", metrics.gauges)
	}
}

func TestHealthCheckElapsedDeadlineUsesCurrentContextError(t *testing.T) {
	client, err := New(context.Background(), Config{
		Name:    "resiliencx",
		Timeout: time.Hour,
	})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx := &changingErrDeadlineContext{
		Context:  context.Background(),
		deadline: time.Now().Add(-time.Second),
		err:      context.Canceled,
	}
	status := client.HealthCheck(ctx)
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if status.Message != context.Canceled.Error() {
		t.Fatalf("expected current context error message, got %q", status.Message)
	}
}

func TestHealthCheckZeroValueClientUnhealthy(t *testing.T) {
	var client Client

	status := client.HealthCheck(context.Background())
	if status.Status != HealthUnhealthy {
		t.Fatalf("expected unhealthy status, got %q", status.Status)
	}
	if status.Name != "resiliencx" {
		t.Fatalf("expected fallback health name, got %q", status.Name)
	}
}

func TestHealthStatusJSONContract(t *testing.T) {
	payload, err := json.Marshal(HealthStatus{
		Name:      "resiliencx",
		Status:    HealthHealthy,
		LatencyMs: 7,
	})
	if err != nil {
		t.Fatalf("marshal health status: %v", err)
	}
	encoded := string(payload)
	for _, field := range []string{"name", "status", "checked_at", "latency_ms"} {
		if !strings.Contains(encoded, `"`+field+`"`) {
			t.Fatalf("expected JSON field %q in %s", field, encoded)
		}
	}
	if strings.Contains(encoded, "CheckedAt") || strings.Contains(encoded, "LatencyMs") {
		t.Fatalf("expected snake_case JSON fields, got %s", encoded)
	}
}

type deadlineOnlyContext struct {
	context.Context
	deadline time.Time
}

func (ctx deadlineOnlyContext) Deadline() (time.Time, bool) {
	return ctx.deadline, true
}

func (ctx deadlineOnlyContext) Err() error {
	return nil
}

type changingErrDeadlineContext struct {
	context.Context
	deadline time.Time
	err      error
	errCalls int
}

func (ctx *changingErrDeadlineContext) Deadline() (time.Time, bool) {
	return ctx.deadline, true
}

func (ctx *changingErrDeadlineContext) Err() error {
	ctx.errCalls++
	if ctx.errCalls == 1 {
		return nil
	}
	return ctx.err
}

func TestHealthCheckUsesInjectedClockForCheckedAtAndLatency(t *testing.T) {
	base := time.Date(2026, 6, 4, 2, 0, 0, 0, time.UTC)
	clk := &sequenceClock{times: []time.Time{
		base,
		base.Add(1234 * time.Millisecond),
	}}
	client, err := New(context.Background(), Config{Name: "templatex"}, withClock(clk))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(context.Background())
	if status.Status != HealthHealthy {
		t.Fatalf("expected healthy status, got %q", status.Status)
	}
	if !status.CheckedAt.Equal(base.Add(1234 * time.Millisecond)) {
		t.Fatalf("checked_at = %s, want %s", status.CheckedAt, base.Add(1234*time.Millisecond))
	}
	if status.LatencyMs != 1234 {
		t.Fatalf("latency_ms = %d, want 1234", status.LatencyMs)
	}
}

func TestHealthCheckDeadlineUsesInjectedClock(t *testing.T) {
	base := time.Date(2026, 6, 4, 2, 0, 0, 0, time.UTC)
	client, err := New(context.Background(), Config{
		Name:    "templatex",
		Timeout: 10 * time.Second,
	}, withClock(&sequenceClock{times: []time.Time{
		base,
		base,
		base.Add(25 * time.Millisecond),
	}}))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	status := client.HealthCheck(deadlineOnlyContext{
		Context:  context.Background(),
		deadline: base.Add(5 * time.Second),
	})
	if status.Status != HealthDegraded {
		t.Fatalf("expected degraded status, got %q", status.Status)
	}
	if status.Metadata["reason"] != "deadline_below_timeout" {
		t.Fatalf("expected deterministic deadline metadata, got %#v", status.Metadata)
	}
	if !status.CheckedAt.Equal(base.Add(25 * time.Millisecond)) {
		t.Fatalf("checked_at = %s, want %s", status.CheckedAt, base.Add(25*time.Millisecond))
	}
	if status.LatencyMs != 25 {
		t.Fatalf("latency_ms = %d, want 25", status.LatencyMs)
	}
}

func TestHealthCheckClockIsClientLocal(t *testing.T) {
	base := time.Date(2026, 6, 4, 2, 0, 0, 0, time.UTC)
	clientA, err := New(context.Background(), Config{Name: "a"}, withClock(&sequenceClock{times: []time.Time{
		base,
		base.Add(time.Millisecond),
	}}))
	if err != nil {
		t.Fatalf("new client a: %v", err)
	}
	clientB, err := New(context.Background(), Config{Name: "b"}, withClock(&sequenceClock{times: []time.Time{
		base.Add(time.Hour),
		base.Add(time.Hour + 2*time.Millisecond),
	}}))
	if err != nil {
		t.Fatalf("new client b: %v", err)
	}

	statusA := clientA.HealthCheck(context.Background())
	statusB := clientB.HealthCheck(context.Background())
	if statusA.LatencyMs != 1 {
		t.Fatalf("client A latency_ms = %d, want 1", statusA.LatencyMs)
	}
	if statusB.LatencyMs != 2 {
		t.Fatalf("client B latency_ms = %d, want 2", statusB.LatencyMs)
	}
	if statusA.CheckedAt.Equal(statusB.CheckedAt) {
		t.Fatalf("expected client-local clocks, got matching checked_at %s", statusA.CheckedAt)
	}
}

type sequenceClock struct {
	times []time.Time
	idx   int
}

func (c *sequenceClock) Now() time.Time {
	if len(c.times) == 0 {
		return time.Time{}
	}
	idx := c.idx
	if idx >= len(c.times) {
		idx = len(c.times) - 1
	} else {
		c.idx++
	}
	return c.times[idx]
}
