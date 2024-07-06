package contexts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewActivityLogContext(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	assert.Equal(t, "operationId", ctx.operationId)
	assert.Equal(t, "callerId", ctx.callerId)
	assert.Equal(t, "resourceCloudId", ctx.resourceCloudId)
	assert.Equal(t, "resourceK8SId", ctx.resourceK8SId)
}

func TestActivityLogContext_ToMap(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	m := ctx.ToMap()
	assert.NotNil(t, m)
	assert.Equal(t, "operationId", m["operation-id"])
	assert.Equal(t, "callerId", m["caller-id"])
	assert.Equal(t, "resourceCloudId", m["resource-cloud-id"])
	assert.Equal(t, "resourceK8SId", m["resource-k8s-id"])
}

func TestActivityLogContext_FromMap(t *testing.T) {
	ctx := NewActivityLogContext("a_operationId", "a_callerId", "a_resourceCloudId", "a_resourceK8SId")
	assert.NotNil(t, ctx)
	m := map[string]interface{}{
		"operation-id":      "operationId",
		"caller-id":         "callerId",
		"resource-cloud-id": "resourceCloudId",
		"resource-k8s-id":   "resourceK8SId",
	}
	ctx.FromMap(m)
	assert.Equal(t, "operationId", ctx.operationId)
	assert.Equal(t, "callerId", ctx.callerId)
	assert.Equal(t, "resourceCloudId", ctx.resourceCloudId)
	assert.Equal(t, "resourceK8SId", ctx.resourceK8SId)
}

func TestActivityLogContext_FromMapMissingFields(t *testing.T) {
	ctx := NewActivityLogContext("a_operationId", "a_callerId", "a_resourceCloudId", "a_resourceK8SId")
	assert.NotNil(t, ctx)
	m := map[string]interface{}{
		"operation-id": "operationId",
	}
	ctx.FromMap(m)
	assert.Equal(t, "operationId", ctx.operationId)
	assert.Equal(t, "a_callerId", ctx.callerId)
	assert.Equal(t, "a_resourceCloudId", ctx.resourceCloudId)
	assert.Equal(t, "a_resourceK8SId", ctx.resourceK8SId)
}

func TestActivityLogContext_Deadline(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	deadline, ok := ctx.Deadline()
	assert.False(t, ok)
	assert.Equal(t, deadline, deadline)
}

func TestActivityLogContext_Done(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	done := ctx.Done()
	assert.Nil(t, done)
}

func TestActivityLogContext_Err(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	err := ctx.Err()
	assert.Nil(t, err)
}

func TestActivityLogContext_Value(t *testing.T) {
	ctx := NewActivityLogContext("operationId", "callerId", "resourceCloudId", "resourceK8SId")
	assert.NotNil(t, ctx)
	assert.Equal(t, "operationId", ctx.Value("operation-id"))
	assert.Equal(t, "callerId", ctx.Value("caller-id"))
	assert.Equal(t, "resourceCloudId", ctx.Value("resource-cloud-id"))
	assert.Equal(t, "resourceK8SId", ctx.Value("resource-k8s-id"))
}

func TestNewDiagnosticLogContext(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	assert.Equal(t, "traceId", ctx.traceId)
	assert.Equal(t, "spanId", ctx.spanId)
	assert.Equal(t, "correlationId", ctx.correlationId)
	assert.Equal(t, "requestId", ctx.requestId)
}

func TestDiagnosticsLogContext_ToMap(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	m := ctx.ToMap()
	assert.NotNil(t, m)
	assert.Equal(t, "traceId", m["trace-id"])
	assert.Equal(t, "spanId", m["span-id"])
	assert.Equal(t, "correlationId", m["correlation-id"])
	assert.Equal(t, "requestId", m["request-id"])
}

func TestDiagnosticsLogContext_FromMap(t *testing.T) {
	ctx := NewDiagnosticLogContext("a_traceId", "a_spanId", "a_correlationId", "a_requestId")
	assert.NotNil(t, ctx)
	m := map[string]interface{}{
		"trace-id":       "traceId",
		"span-id":        "spanId",
		"correlation-id": "correlationId",
		"request-id":     "requestId",
	}
	ctx.FromMap(m)
	assert.Equal(t, "traceId", ctx.traceId)
	assert.Equal(t, "spanId", ctx.spanId)
	assert.Equal(t, "correlationId", ctx.correlationId)
	assert.Equal(t, "requestId", ctx.requestId)
}

func TestDiagnosticsLogContext_FromMapMissingFields(t *testing.T) {
	ctx := NewDiagnosticLogContext("a_traceId", "a_spanId", "a_correlationId", "a_requestId")
	assert.NotNil(t, ctx)
	m := map[string]interface{}{
		"trace-id": "traceId",
	}
	ctx.FromMap(m)
	assert.Equal(t, "traceId", ctx.traceId)
	assert.Equal(t, "a_spanId", ctx.spanId)
	assert.Equal(t, "a_correlationId", ctx.correlationId)
	assert.Equal(t, "a_requestId", ctx.requestId)
}

func TestDiagnosticsLogContext_Deadline(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	deadline, ok := ctx.Deadline()
	assert.False(t, ok)
	assert.Equal(t, deadline, deadline)
}

func TestDiagnosticsLogContext_Done(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	done := ctx.Done()
	assert.Nil(t, done)
}

func TestDiagnosticsLogContext_Err(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	err := ctx.Err()
	assert.Nil(t, err)
}

func TestDiagnosticsLogContext_Value(t *testing.T) {
	ctx := NewDiagnosticLogContext("traceId", "spanId", "correlationId", "requestId")
	assert.NotNil(t, ctx)
	assert.Equal(t, "traceId", ctx.Value("trace-id"))
	assert.Equal(t, "spanId", ctx.Value("span-id"))
	assert.Equal(t, "correlationId", ctx.Value("correlation-id"))
	assert.Equal(t, "requestId", ctx.Value("request-id"))
}
